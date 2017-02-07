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

type cloudVolume struct {
	Id       string `json:"id"`
	Status   string `json:"status"`
	Type     string `json:"type"`
	Size     int    `json:"size"`
	Region   string `json:"region"`
	Bootable bool   `json:"bootable"`
}

func resourceCloudVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudVolumeCreate,
		Read:   resourceCloudVolumeRead,
		Delete: resourceCloudVolumeDelete,

		Schema: map[string]*schema.Schema{
			"project": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": {
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
			"size": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"volume_id": {
				Type:     schema.TypeString,
				Computed: true,
				Required: false,
			},
		},
	}
}

func resourceCloudVolumeRead(d *schema.ResourceData, meta interface{}) error {
	Config := meta.(*Config)
	resp := cloudInstance{}
	uri := fmt.Sprintf("/cloud/project/%s/volume/%s", d.Get("project").(string), d.Get("volume_id").(string))
	log.Printf("[DEBUG] Calling OVH API")
	err := Config.OVHClient.Get(uri, &resp)
	if err != nil {
		return fmt.Errorf("[ERROR] calling %s:\n\t %q", uri, err)
	}
	return nil
}

func resourceCloudVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	Config := meta.(*Config)

	endpoint := fmt.Sprintf("/cloud/project/%s/volume", d.Get("project").(string))
	response := cloudInstance{}
	request := cloudVolume{
		Region:   d.Get("region").(string),
		Size:     d.Get("size").(int),
		Type:     d.Get("type").(string),
		Bootable: false,
	}

	log.Printf("[DEBUG] OVH API Call :\n\n%v", request)
	err := Config.OVHClient.Post(endpoint, request, &response)
	if err != nil {
		return fmt.Errorf("[ERROR] calling %s:\n\t %q", endpoint, err)
	}

	stateConf := &resource.StateChangeConf{
		Target: []string{"available"},
		Refresh: func() (interface{}, string, error) {
			resp := cloudInstance{}
			uri := fmt.Sprintf("/cloud/project/%s/volume/%s", d.Get("project").(string), response.Id)
			log.Printf("[DEBUG] Calling OVH API")
			err := Config.OVHClient.Get(uri, &resp)
			if err != nil {
				log.Printf("[DEBUG] Error in call %s", err)
				return resp.Id, "", err
			}
			log.Printf("[DEBUG] Pending volume creation for %s with status: %s", resp.Id, resp.Status)
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

	d.Set("volume_id", response.Id)
	d.SetId(fmt.Sprintf("ovh_cloud_project_%s_volume_%s", d.Get("project"), response.Id))

	return resourceCloudVolumeRead(d, meta)
}

func resourceCloudVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	Config := meta.(*Config)

	endpoint := fmt.Sprintf("/cloud/project/%s/volume/%s", d.Get("project").(string), d.Get("volume_id").(string))
	response := cloudInstance{}
	log.Printf("[DEBUG] sending POST to OVH API")
	err := Config.OVHClient.Delete(endpoint, &response)
	if err != nil {
		return fmt.Errorf("[ERROR] calling %s:\n\t %q", endpoint, err)
	}

	log.Printf("[DEBUG] Removed instance %s from project %s)", d.Get("instance_id"), d.Get("project"))

	return nil
}
