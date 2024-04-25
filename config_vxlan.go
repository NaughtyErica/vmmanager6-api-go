package vmmanager6

import (
	"fmt"
	"encoding/json"
)

type VxLANipnets struct {
	Id	int	`json:"id"`
	Name	string	`json:"name"`
	Gateway string  `json:"gateway"`
}

type ConfigNewVxLAN struct {
	Name		string			`json:"name"`
	Account 	int 			`json:"account"`
	Clusters	[]int 			`json:"clusters"`
	Comment		string			`json:"comment"`
	Ips 		[]VxLANipnets   `json:"ipnets"`
}

type ConfigVxLAN struct {
	Id 			int 			`json:"id"`
	Name 		string			`json:"name"`
	Account     ConfigAccount   `json:"account"`
	Comment     string 			`json:"comment"`
	Ips 		[]VxLANipnets   `json:"ipnets"`
	Ippool		int 			`json:"ippool"`
}

func (config ConfigNewVxLAN) CreateVxLAN(client *Client) (vmid string, err error) {
	vmid, err = client.AccountAddVxLAN(config)
	if err != nil {
                return "", fmt.Errorf("error creating VxLAN: %v (params: %v)", err, config)
        }

	return
}

func NewConfigVxLANFromApi(id string, client *Client) (config *ConfigVxLAN, err error) {
	api_config, err := client.GetVxLANInfo(id)
	j, err := json.Marshal(api_config)
	err = json.Unmarshal(j, &config)
	return
}

