package vmmanager6

import (
	"fmt"
	"encoding/json"
//	"log"
)

type ConfigDisk struct {
	Size		[]int	    `json:"disk_mib"`
	Id		[]int	    `json:"id"`
}
type ClusterConfig struct {
	Id		int	    `json:"id"`
	DatacenterType  string	    `json:"datacenter_type"`
	Name		string	    `json:"name"`
	Type		string	    `json:"virtualization_type"`
}
type NodeConfig struct {
	Id		int	    `json:"id"`
	Name		string	    `json:"name"`
}
type AccountConfig struct {
	Email		string	    `json:"email"`
	Id		int	    `json:"id"`
}
type OsConfig struct {
	Id		int	    `json:"id"`
}
type IpConfig struct {
	Domain		string		`json:"domain"`
	Family		int 		`json:"family"`
	Gateway 	string		`json:"gateway"`
	Id  		int 		`json:"id"`
	Addr 		string		`json:"ip_addr"`
	Mask	 	string		`json:"mask"`
	NetId		int 		`json:"network"`
}
type Ipv4Config struct {
	Interface	string	    `json:"interface"`
	Ip		string	    `json:"ip"`
}
type RecipeParamsConfig struct {
	Name		string		`json:"name"`
	Value		string		`json:"value"`
}
type RecipeConfig struct {
	Recipe 		int 		`json:"recipe"`
	Params 		[]RecipeParamsConfig	`json:"recipe_params"`
}
// ConfigQemu - VMmanager6 API QEMU options
type ConfigQemu struct {
	Name            string      `json:"name"`
	Description     string      `json:"comment"`
	QemuCores       int         `json:"cpu_number"`
	Memory          int         `json:"ram_mib"`
	QemuDisks       ConfigDisk  `json:"disk"`
	Cluster		ClusterConfig	`json:"cluster"`
	Node		NodeConfig		`json:"node"`
	Account		AccountConfig	`json:"account"`
	Domain		string	    `json:"domain"`
	Os		OsConfig    `json:"os"`
	Anti_spoofing		bool	`json:"anti_spoofing"`
	IPv4		[]Ipv4Config	`json:"ip4"`
}
type UpdateConfigQemu struct {
	Name            string      `json:"name"`
	Description     string      `json:"comment"`
}
type ConfigNewQemu struct {
	Name            string      	`json:"name"`
	Description     string      	`json:"comment"`
	QemuCores       int         	`json:"cpu_number"`
	Memory          int         	`json:"ram_mib"`
	QemuDisks       int	    		`json:"hdd_mib"`
	Cluster			int	    		`json:"cluster"`
	Account			int	    		`json:"account"`
	Node			int				`json:"node"`
	Domain			string	    	`json:"domain"`
	Os				int         	`json:"os"`
	Anti_spoofing		bool	`json:"anti_spoofing"`
	IPv4			int	    		`json:"ipv4_number"`
	IPv4Pools		[]int	    	`json:"ipv4_pool"`
	Password		string	    	`json:"password"`
	CpuMode			string		`json:"cpu_mode"`
	Recipes 		[]RecipeConfig 	`json:"recipe_list"`
	CustomInterfaces []interface{}  `json:"custom_interfaces"`
	Vxlans			[]interface{}   `json:"vxlan"`
	Preset		int	`json:"preset"`
}
type ReinstallOS struct {
	Id		int         `json:"os"`
	Password	string      `json:"password"`
	EmailMode	string      `json:"send_email_mode"`
}
type ResourcesQemu struct {
    Cores		int	`json:"cpu_number"`
    Memory		int	`json:"ram_mib"`
    CpuMode		string  `json:"cpu_mode"`
}
// CreateVm - Tell VMmanager 6 API to make the VM
func (config ConfigNewQemu) CreateVm(client *Client) (vmid int, err error) {
	vmid, err = client.CreateQemuVm(config)
	if err != nil {
                return 0, fmt.Errorf("error creating VM: %v (params: %v)", err, config)
        }

	return
}

func NewConfigQemuFromApi(vmr *VmRef, client *Client) (config *ConfigQemu, err error) {
    var vmConfig map[string]interface{}
	vmConfig, err = client.GetVmInfo(vmr)
	j, err := json.Marshal(vmConfig)
	err = json.Unmarshal(j, &config)
	return
}

func NewConfigQemuIpsFromApi(vmr *VmRef, client *Client) (config []*IpConfig, err error) {
    var ipConfig []interface{}
    ipConfig, err = client.GetVmIpsInfo(vmr)
    j, err := json.Marshal(ipConfig)
    err = json.Unmarshal(j, &config)
    return
}

func (config ResourcesQemu) UpdateResources(vmr *VmRef, client *Client) (err error) {
	err = client.UpdateQemuResources(vmr, config)
	if err != nil {
                return fmt.Errorf("error updating resources of VM id %v: %v (params: %v)", vmr.vmId, err, config)
	}
	return
}

func (config ConfigDisk) UpdateDisk(client *Client) (err error) {
	err = client.UpdateQemuDisk(config)
	if err != nil {
                return fmt.Errorf("error updating disk of VM: %v (params: %v)", err, config)
	}
	return
}

func (config UpdateConfigQemu) UpdateConfig(vmr *VmRef, client *Client) (err error) {
	err = client.UpdateQemuConfig(vmr, config)
	if err != nil {
                return fmt.Errorf("error updating config of VM id %v: %v (params: %v)", vmr.vmId, err, config)
	}
	return
}

func (config ReinstallOS) ReinstallOS(vmr *VmRef, client *Client) (err error) {
	err = client.ReinstallQemu(vmr, config)
	if err != nil {
                return fmt.Errorf("error reinstalling of VM id %v: %v (params: %v)", vmr.vmId, err, config)
	}
	return
}
