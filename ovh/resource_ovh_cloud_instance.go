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
	NetworkId	string `json:"networkId"`
	Ip		string `json:"ip"`
	Version		int `json:"version"`
	Type		string `json:"type"`
}

type cloudInstance struct {
	Id		string `json:"id"`
	Status		string `json:"status"`
        IpAddresses	[]cloudInstanceIpAddress `json:"ipAddresses"`
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
				Type:		schema.TypeString,
				Optional:	true,
				ForceNew:	true,
			},
			"ssh_key": {
				Type:     schema.TypeString,
				Required: true,
                                ForceNew: true,
			},
			"ip": {
				Type:		schema.TypeString,
				Computed:	true,
				Required:	false,
			},
			"instance_id": {
				Type:		schema.TypeString,
				Computed:	true,
				Required:	false,
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

func resourceCloudInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	Config := meta.(*Config)

		endpoint := fmt.Sprintf("/cloud/project/%s/instance", d.Get("project").(string))
		response := cloudInstance{}
		request := map[string]string{
			"flavorId":	d.Get("flavor").(string),
			"imageId":	d.Get("image").(string),
			"name":		d.Get("name").(string),
			"region":	d.Get("region").(string),
			"sshKey":	d.Get("ssh_key").(string),
			"userData":	d.Get("userdata").(string),
		}

		log.Printf("[DEBUG] OVH API Call :\n\n%v", request)
		err := Config.OVHClient.Post(endpoint, request, &response)
		if err != nil {
			return fmt.Errorf("[ERROR] calling %s:\n\t %q", endpoint, err)
		}


		stateConf := &resource.StateChangeConf{
			Pending:    []string{"BUILD", "BUILDING"},
	                Target:     []string{"ACTIVE"},
	                Refresh:    func() (interface{}, string, error) {
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

	log.Printf("[DEBUG] Removed instance %s from project %s)", d.Get("instance_id") ,d.Get("project"))

	return nil
}

