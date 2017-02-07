package ovh

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	//	"github.com/ovh/go-ovh/ovh"
	"log"
	"time"
	//	"io/ioutil"
)

/*
type cloudInstance struct {
	serviceName	string
	flavorId	string
	groupId		string
	imageId		string
	monthlyBilling	string
	name		string
	networks	string
	region		string
	sshKeyId	string
	userData	string
	status		string
	created		string
	id		string
	ipAddresses	[]string
} */

type cloudInstanceIpAddress struct {
	NetworkId string `json:"networkId"`
	Ip        string `json:"ip"`
	Version   int    `json:"version"`
	Type      string `json:"type"`
}

type cloudInstance struct {
	Id          string                   `json:"id"`
	Status      string                   `json:"status"`
	IpAddresses []cloudInstanceIpAddress `json:"ipAddresses"`
}

func resourceCloudInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudInstanceCreate,
		Read:   resourceCloudInstanceRead,
		Delete: resourceCloudInstanceDelete,

		Schema: map[string]*schema.Schema{
			"project": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"flavor": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"image": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"userdata": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"ssh_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ip": {
				Type:     schema.TypeString,
				Computed: true,
				Required: false,
			},
			"instance_id": {
				Type:     schema.TypeString,
				Computed: true,
				Required: false,
			},
		},
	}
}

func resourceCloudInstanceRead(d *schema.ResourceData, meta interface{}) error {
	Config := meta.(*Config)
	resp := cloudInstance{}
	uri := fmt.Sprintf("/cloud/project/%s/instance/%s", d.Get("project").(string), d.Get("instance_id").(string))
	log.Printf("[DEBUG] Calling OVH API")
	err := Config.OVHClient.Get(uri, &resp)
	if err != nil {
		return fmt.Errorf("[ERROR] calling %s:\n\t %q", uri, err)
	}
	d.Set("ip", resp.IpAddresses[0].Ip)
	return nil
}

// cloudInstanceFlavor defines vms parameters for provisioning
type cloudInstanceFlavor struct {
	OutboundBandwidth int    `json:"outboundBandwidth"`
	Disk              int    `json:"disk"`
	Region            string `json:"region"`
	Name              string `json:"name"`
	InboundBandwidth  int    `json:"inboundBandwidth"`
	Id                string `json:"id"`
	Vcpus             int    `json:"vcpus"`
	Type              string `json:"type"`
	OsType            string `json:"osType"`
	Available         bool   `json:"available"`
	Ram               int    `json:"ram"`
}

// Get flavor object based on name and region
func getFlavor(project string, name string, region string, meta interface{}) (flavor cloudInstanceFlavor, err error) {
	Config := meta.(*Config)
	endpoint := fmt.Sprintf("/cloud/project/%s/flavor", project)
	response := []cloudInstanceFlavor{}

	err = Config.OVHClient.Get(endpoint, &response)
	if err != nil {
		return
	}

	// Look for available flavor with matching name/region
	for _, flavor := range response {
		if flavor.Name == name && flavor.Region == region && flavor.Available {
			return flavor, nil
		}
	}

	// No flavor found for given criteria
	err = fmt.Errorf("no flavor found")
	return
}

type cloudInstanceImage struct {
	Size         float32 `json:"size"`
	CreationDate string  `json:"creationDate"`
	User         string  `json:"user"`
	Id           string  `json:"id"`
	Visibility   string  `json:"visibility"`
	Status       string  `json:"status"`
	Type         string  `json:"type"`
	Region       string  `json:"region"`
	Name         string  `json:"name"`
	MinRam       int     `json:"minRam"`
	MinDisk      int     `json:"minDisk"`
}

func getImage(project string, name string, region string, meta interface{}) (image cloudInstanceImage, err error) {
	Config := meta.(*Config)
	endpoint := fmt.Sprintf("/cloud/project/%s/image?region=%s", project, region)
	response := []cloudInstanceImage{}

	err = Config.OVHClient.Get(endpoint, &response)
	if err != nil {
		return
	}

	// Look for available flavor with matching name/region
	for _, image := range response {
		if image.Name == name && image.Status == "active" {
			return image, nil
		}
	}

	// No flavor found for given criteria
	err = fmt.Errorf("no flavor found")
	return
}

func resourceCloudInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	Config := meta.(*Config)

	flavor, err := getFlavor(d.Get("project").(string), d.Get("flavor").(string), d.Get("region").(string), meta)
	if err != nil {
		panic("could not establish flavor Id")
	}

	image, err := getImage(d.Get("project").(string), d.Get("image").(string), d.Get("region").(string), meta)
	if err != nil {
		panic("could not establish image Id")
	}

	endpoint := fmt.Sprintf("/cloud/project/%s/instance", d.Get("project").(string))
	response := cloudInstance{}
	request := map[string]string{
		"flavorId": flavor.Id,
		"imageId":  image.Id,
		"name":     d.Get("name").(string),
		"region":   d.Get("region").(string),
		"sshKey":   d.Get("ssh_key").(string),
		"userData": d.Get("userdata").(string),
	}

	log.Printf("[DEBUG] OVH API Call :\n\n%v", request)
	err = Config.OVHClient.Post(endpoint, request, &response)
	if err != nil {
		return fmt.Errorf("[ERROR] calling %s:\n\t %q", endpoint, err)
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{"BUILD", "BUILDING"},
		Target:  []string{"ACTIVE"},
		Refresh: func() (interface{}, string, error) {
			resp := cloudInstance{}
			uri := fmt.Sprintf("/cloud/project/%s/instance/%s", d.Get("project").(string), response.Id)
			log.Printf("[DEBUG] Calling OVH API")
			err := Config.OVHClient.Get(uri, &resp)
			if err != nil {
				log.Printf("[DEBUG] Error in call %s", err)
				return resp.Id, "", err
			}
			log.Printf("[DEBUG] Pending instance creation for %s with status: %s", resp.Id, resp.Status)
			return resp.Id, resp.Status, nil

		},
		Timeout:    15 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	log.Printf("[DEBUG] Entering waitstate")
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[ERROR] waiting for state")
	}

	d.Set("instance_id", response.Id)
	d.SetId(fmt.Sprintf("ovh_cloud_project_%s_instance_%s", d.Get("project"), response.Id))

	return resourceCloudInstanceRead(d, meta)
}

func resourceCloudInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	Config := meta.(*Config)

	endpoint := fmt.Sprintf("/cloud/project/%s/instance/%s", d.Get("project").(string), d.Get("instance_id").(string))
	response := cloudInstance{}
	log.Printf("[DEBUG] sending POST to OVH API")
	err := Config.OVHClient.Delete(endpoint, &response)
	if err != nil {
		return fmt.Errorf("[ERROR] calling %s:\n\t %q", endpoint, err)
	}

	log.Printf("[DEBUG] Removed instance %s from project %s)", d.Get("instance_id"), d.Get("project"))

	return nil
}
