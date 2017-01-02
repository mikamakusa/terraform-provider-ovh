package ovh

//import (
//	"github.com/ovh/go-ovh/ovh"
//)

type ovhTask struct {
	CreationDate	string `json:"creationDate"`
        Status          string `json:"status"`
        Action          string `json:"action"`
        Id		int    `json:"id"`
	EndpointTemplate	string
	Endpoint	string
}

type ovhTaskResponse struct {
	CreationDate	string `json:"creationDate"`
        Status          string `json:"status"`
        Action          string `json:"action"`
        Id		int    `json:"id"`
}

// func (task ovhTask) IsFinished() (bool) {
// }


