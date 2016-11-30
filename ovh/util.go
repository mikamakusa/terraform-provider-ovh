package ovh

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
)

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

type PublicCloudInstance struct {
	Status string `json:"status"`
	Name   string `json:"name"`
	Region string `json:"region"`
	//Image        string        `json:"image"`
	Created        string `json:"created"`
	SSHKey         string `json:"sshKey"`
	MonthlyBilling string `json:"monthlyBilling"`
	//IPAddresses  string        `json:"ipAddresses"`
	Id string `json:"id"`
	//Flavor       string        `json:"Flavor"`
}

func GetPublicCloudInstance(d *schema.ResourceData, Config *Config) (PublicCloudInstance, error) {
	var err error

	response := PublicCloudInstance{}

	endpoint := fmt.Sprintf("/cloud/project/%s/instance/%s", d.Get("project_id").(string), d.Get("instance_id").(string))

	err = Config.OVHClient.Get(endpoint, &response)
	if err != nil {
		return PublicCloudInstance{}, fmt.Errorf("[ERROR] calling %s:\n\t %q", endpoint, err)
	}

	return response, nil
}