package ovh

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/ovh/go-ovh/ovh"
	"log"
	"time"
)

type ipLoadbalancingBackend struct {
	Backend       string `json:"backend"`
	Probe         string `json:"probe"`
	Weight        string `json:"weight"`
	Zone          string `json:"zone"`
}

type ipLoadbalancingBackendTask struct {
	*ovhTask
}

type ipLoadbalancingBackendTaskResponse struct {
	*ovhTaskResponse
}

func resourceIpLoadbalancingBackend() *schema.Resource {
	return &schema.Resource{
		Create: resourceIpLoadbalancingBackendCreate,
		Read:   resourceIpLoadbalancingBackendRead,
		Delete: resourceIpLoadbalancingBackendDelete,

		Schema: map[string]*schema.Schema{
			"service_name": {
				Type:     schema.TypeString,
                                Required: true,
				ForceNew: true,
			},
			"backend_ip": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"probe": {
				Type:     schema.TypeString,
				Required: true,
                                ForceNew: true,
			},
		},
	}
}

func resourceIpLoadbalancingBackendRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceIpLoadbalancingBackendCreate(d *schema.ResourceData, meta interface{}) error {
	Config := meta.(*Config)

	endpoint := fmt.Sprintf("/ip/loadBalancing/%s/backend", d.Get("service_name").(string))
	response := ovhTask{ EndpointTemplate: "/ip/loadBalancing/{{service_name}}/task/{{task_id}}" }
	request := map[string]string{
		"ipBackend": d.Get("backend_ip").(string),
		"probe": d.Get("probe").(string),
	}

	err := Config.OVHClient.Post(endpoint, request, &response)
	if err != nil {
		return fmt.Errorf("[ERROR] calling %s:\n\t %q", endpoint, err)
	}

	stateConf := &resource.StateChangeConf{
                Pending:    []string{"init", "todo", "doing"},
                Target:     []string{"completed"},
                Refresh:    func() (interface{}, string, error) {
			serviceName := d.Get("service_name").(string)
	                r := ipLoadbalancingBackendTaskResponse{}
	                endpoint := fmt.Sprintf("/ip/loadBalancing/%s/task/%d", serviceName, response.Id)
	                err := Config.OVHClient.Get(endpoint, &r)
	                if err != nil {
	                        if err.(*ovh.APIError).Code == 404 {
	                                log.Printf("[DEBUG] Task id %d on %s completed", response.Id, serviceName)
	                                return response.Id, "completed", nil
	                        } else {
	                                return response.Id, "", err
	                        }
	                }
	                log.Printf("[DEBUG] Pending Task id %d on ip loadbalancing backend %s status: %s", r.Id, serviceName, r.Status)
	                return response.Id, r.Status, nil
	        },
                Timeout:    15 * time.Minute,
                Delay:      15 * time.Second,
                MinTimeout: 5 * time.Second,
        }

        _, err = stateConf.WaitForState()

	d.SetId(fmt.Sprintf("ip_loadbalancer_%s_backend_%s", d.Get("service_name"), d.Get("backend_ip")))
	log.Printf("[DEBUG] Attached backend %s to service %s)", d.Get("backend_ip") ,d.Get("service_name"),)

	return nil
}

func resourceIpLoadbalancingBackendDelete(d *schema.ResourceData, meta interface{}) error {
	Config := meta.(*Config)

	endpoint := fmt.Sprintf("/ip/loadBalancing/%s/backend/%s", d.Get("service_name").(string), d.Get("backend_ip").(string))
        response := ipLoadbalancingBackendTask{}

        log.Printf("[DEBUG] sending POST to OVH API")
	err := Config.OVHClient.Delete(endpoint, &response)
	if err != nil {
		return fmt.Errorf("[ERROR] calling %s:\n\t %q", endpoint, err)
	}

	log.Printf("[DEBUG] Removed backend %s from service %s)", d.Get("backend_ip") ,d.Get("service_name"),)

	return nil
}

