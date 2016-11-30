package ovh

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/ovh/go-ovh/ovh"
	"log"
	"time"
)

type PublicCloudFailoverIp struct {
	ContinentCode string `json:"continentCode"`
	Progress      int    `json:"progress"`
	Status        string `json:"status"`
	IP            string `json:"ip"`
	RoutedTo      string `json:"routedTo"`
	SubType       string `json:"subType"`
	Id            string `json:"id"`
	Block         string `json:"block"`
	GeoLocation   string `json:"geoloc"`
	Regions       []string
}

func resourcePublicCloudFailoverIp() *schema.Resource {
	return &schema.Resource{
		Create: resourcePublicCloudFailoverIpCreate,
		Read:   resourcePublicCloudFailoverIpRead,
		Delete: resourcePublicCloudFailoverIpDelete,

		Schema: map[string]*schema.Schema{
			"ip_address": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"ip_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"instance_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_PUBLIC_CLOUD_PROJECT_ID", ""),
			},
		},
	}
}

func resourcePublicCloudFailoverIpCreate(d *schema.ResourceData, meta interface{}) error {
	Config := meta.(*Config)

	ip, err := publicCloudGetFailoverIpFromConfig(d, Config)
	if err != nil {
		return err
	}

	instance, err := GetPublicCloudInstance(d, Config)
	if err != nil {
		return err
	}

	if !stringInSlice(instance.Region, ip.Regions) {
		return fmt.Errorf("[ERROR] IP %s cannot be used in region %s", ip.IP, instance.Region)
	}

	endpoint := fmt.Sprintf("/cloud/project/%s/ip/failover/%s/attach", d.Get("project_id").(string), ip.Id)
	response := PublicCloudFailoverIp{}
	request := map[string]string{
		"instanceId": d.Get("instance_id").(string),
	}

	err = Config.OVHClient.Post(endpoint, request, &response)
	if err != nil {
		return fmt.Errorf("[ERROR] calling %s:\n\t %q", endpoint, err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"operationPending"},
		Target:     []string{"ok"},
		Refresh:    publicCloudFailoverIpRefreshFunc(Config.OVHClient, d.Get("project_id").(string), ip),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[ERROR] Waiting for failover ip (%s, %s): %s", ip.Id, ip.IP, err)
	}

	d.SetId(fmt.Sprintf("failover_ip_%s-instance_%s", ip.Id, instance.Id))

	log.Printf("[DEBUG] Attatched failover ip (%s, %s)", ip.Id, ip.IP)

	return resourcePublicCloudFailoverIpRead(d, meta)
}

func resourcePublicCloudFailoverIpRead(d *schema.ResourceData, meta interface{}) error {
	Config := meta.(*Config)

	ip, err := publicCloudGetFailoverIpFromConfig(d, Config)
	if err != nil {
		return err
	}

	d.Set("ip_address", ip.IP)
	d.Set("ip_id", ip.Id)
	d.Set("instance_id", ip.RoutedTo)

	return nil
}

//noinspection GoUnusedParameter
func resourcePublicCloudFailoverIpDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func publicCloudFailoverIpRefreshFunc(c *ovh.Client, projectId string, ip PublicCloudFailoverIp) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r := &PublicCloudFailoverIp{}
		endpoint := fmt.Sprintf("/cloud/project/%s/ip/failover/%s", projectId, ip.Id)
		err := c.Get(endpoint, r)
		if err != nil {
			return r, "", err
		}

		log.Printf("[DEBUG] Pending failover ip: %s", r)
		return r, r.Status, nil
	}
}

func publicCloudGetFailoverIpFromConfig(d *schema.ResourceData, c *Config) (PublicCloudFailoverIp, error) {
	projectId := d.Get("project_id").(string)

	if ipId := d.Get("ip_id").(string); ipId != "" {
		return PublicCloudGetFailoverIpById(c.OVHClient, projectId, ipId)
	}

	if ipAddress := d.Get("ip_address").(string); ipAddress != "" {
		return PublicCloudGetFailoverIpByAddress(c.OVHClient, projectId, ipAddress)
	}

	//noinspection GoPlaceholderCount
	return PublicCloudFailoverIp{}, fmt.Errorf("You must specify a ip_address or ip_id.")
}

func PublicCloudGetFailoverIpById(c *ovh.Client, projectID string, ipId string) (PublicCloudFailoverIp, error) {
	endpoint := fmt.Sprintf("/cloud/project/%s/ip/failover/%s", projectID, ipId)
	ip := PublicCloudFailoverIp{}

	err := c.Get(endpoint, &ip)
	if err != nil {
		return PublicCloudFailoverIp{}, fmt.Errorf("[ERROR] calling %s:\n\t %q", endpoint, err)
	}

	return publicCloudGetFailoverIpRegions(ip), nil
}

func PublicCloudGetFailoverIpByAddress(c *ovh.Client, projectID string, ipAddress string) (PublicCloudFailoverIp, error) {
	endpoint := fmt.Sprintf("/cloud/project/%s/ip/failover", projectID)
	response := []PublicCloudFailoverIp{}

	err := c.Get(endpoint, &response)
	if err != nil {
		return PublicCloudFailoverIp{}, fmt.Errorf("[ERROR] calling %s:\n\t %q", endpoint, err)
	}

	for i := 0; i < len(response); i++ {
		if response[i].IP == ipAddress {
			return publicCloudGetFailoverIpRegions(response[i]), nil
		}
	}

	return PublicCloudFailoverIp{}, fmt.Errorf("[ERROR] IP Address does not exist: %s", ipAddress)
}

func publicCloudGetFailoverIpRegions(ip PublicCloudFailoverIp) PublicCloudFailoverIp {
	ipRegions := map[string][]string{
		"BE": {"GRA1", "SBG1"},
		"CA": {"BHS1"},
		"CZ": {"GRA1", "SBG1"},
		"FI": {"GRA1", "SBG1"},
		"FR": {"GRA1", "SBG1"},
		"DE": {"GRA1", "SBG1"},
		"IE": {"GRA1", "SBG1"},
		"IT": {"GRA1", "SBG1"},
		"LT": {"GRA1", "SBG1"},
		"NL": {"GRA1", "SBG1"},
		"PL": {"GRA1", "SBG1"},
		"PT": {"GRA1", "SBG1"},
		"ES": {"GRA1", "SBG1"},
		"UK": {"GRA1", "SBG1"},
		"US": {"BHS1"},
	}

	ip.Regions = ipRegions[ip.GeoLocation]

	return ip
}
