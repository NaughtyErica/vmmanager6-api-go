package vmmanager6

import (
	"encoding/json"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"
)

const exitStatusSuccess = "complete"

// TaskStatusCheckInterval - time between async checks in seconds
const TaskStatusCheckInterval = 5

type Client struct {
	session		*Session
	ApiUrl		string
	Username	string
	Password	string
	TaskTimeout	int
}
type VmRef struct {
	vmId	int
}
// Create client from config
func NewClient(apiUrl string, hclient *http.Client, tls *tls.Config, taskTimeout int) (client *Client, err error) {
	var sess *Session
	sess, err = NewSession(apiUrl, hclient, tls)
	if err == nil {
		client = &Client{session: sess, ApiUrl: apiUrl, TaskTimeout: taskTimeout}
	}
	return client, err
}
// Update token in client config
func (c *Client) SetAPIToken(token string) {
	c.session.SetAPIToken(token)
}
// Login to VMmanager 6
func (c *Client) Login(username string, password string) (err error) {
	c.Username = username
	c.Password = password
	return c.session.Login(username, password)
}
// Send GET request with retries
func (c *Client) GetJsonRetryable(url string, data *map[string]interface{}, tries int) error {
	var statErr error
	for ii := 0; ii < tries; ii++ {
		_, statErr = c.session.GetJSON(url, nil, nil, data)
		if statErr == nil {
			return nil
		}
		log.Printf("[DEBUG][GetJsonRetryable] Sleeping for %d seconds before asking url %s", ii+1, url)
		time.Sleep(time.Duration(ii+1) * time.Second)
	}
	return statErr
}
// Get list of VMmanager nodes
func (c *Client) GetNodeList() (list map[string]interface{}, err error) {
	err = c.GetJsonRetryable("/vm/v3/node", &list, 3)
	return
}
// Get list of VMmanager vms
func (c *Client) GetVmList() (list map[string]interface{}, err error) {
	err = c.GetJsonRetryable("/vm/v3/host", &list, 3)
	return
}
// Delete URL from session
func (c *Client) DeleteUrl(url string) (err error) {
	_, err = c.session.Delete(url, nil, nil)
	return
}
// Create VM ref object from id
func NewVmRef(vmId int) (vmr *VmRef) {
        vmr = &VmRef{vmId: vmId}
        return
}
// Get VM info
func (c *Client) GetVmInfo(vmr *VmRef) (vmInfo map[string]interface{}, err error) {
	var vmlist map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/vm/v3/host?where=id+EQ+%v", vmr.vmId), &vmlist, 3)
	if err != nil {
		return nil, err
	}
	if len(vmlist["list"].([]interface{})) == 0 {
		return nil, fmt.Errorf("can't find vm id %v", vmr.vmId)
	}
	vms := vmlist["list"].([]interface{})
	vmInfo = vms[0].(map[string]interface{})
        return
}
// Get state of VM
func (c *Client) GetVmState(vmr *VmRef) (vmState string, err error) {
	vm, err := c.GetVmInfo(vmr)
        if err != nil {
                return "", err
        }
        if vm["state"] == nil {
		return "", fmt.Errorf("vm STATE not readable")
        }
        vmState = vm["state"].(string)
        return
}
// Create Qemu VM
func (c *Client) CreateQemuVm(vmParams ConfigNewQemu) (vmid int, err error) {
		var data map[string]interface{}
		var config map[string]interface{}
		config_json, _ := json.Marshal(vmParams)
		err = json.Unmarshal(config_json, &config)
	log.Printf(">>> JSON %#v", config)
	// Check if "disks" key exists.
	if disks, ok := config["disks"].([]interface{}); ok {
		// Iterate over each disk in the "disks" array.
		for _, disk := range disks {
			if d, ok := disk.(map[string]interface{}); ok {
				// Check if "size_mib" key exists and is 0.
				if size, ok := d["size_mib"].(int); ok && size == 0 {
					// If "size_mib" is 0, remove the "disks" key from the config
					delete(config, "disk")
					break
				}
			}
		}
	}
	if config["node"].(float64) == 0 {
		delete(config, "node")
	}
	// if len(config["disks"].([]interface{})) > 0 {
	// 	delete(config, "disk")
	// }
	if config["preset"].(float64) > 0 {
		delete(config, "cpu_number")
		delete(config, "ram_mib")
		delete(config, "hdd_mib")
		delete(config, "cpu_mode")
	}else{
		delete(config, "preset")
	}
    if config["ipv4_number"].(float64) == 0 {
		delete(config, "ipv4_number")
	}
	if config["ipv4_pool"] == nil {
		delete(config, "ipv4_pool")
	}
	if config["recipe_list"] == nil {
		delete(config, "recipe_list")
	}
	if config["custom_interfaces"] == nil || len(config["custom_interfaces"].([]interface{})) == 0 {
		delete(config, "custom_interfaces")
	}else{
		cis := config["custom_interfaces"].([]interface{})
		var cis_new []interface{}
		for _, ci := range cis {
			ci_e := ci.(map[string]interface{})
			if ci_e["ip_name"].(string) != "" {
				delete(ci_e, "ippool")
			}else{
				if ci_e["ippool"].(float64) > 0 {
					delete(ci_e, "ip_name")
				}
			}
			cis_new = append(cis_new, ci_e)
		}
		config["custom_interfaces"] = cis_new
	}
	if config["vxlan"] == nil || len(config["vxlan"].([]interface{})) == 0 {
		delete(config, "vxlan")
	}

        _, err = c.session.PostJSON("/vm/v3/host", nil, nil, &config, &data)
        if err != nil {
                return 0, err
        }
	if data == nil {
		return 0, fmt.Errorf("Can't create VM with params %v", vmParams)
	}
        err = c.WaitForCompletion(data)
	vmid = int(data["id"].(float64))
        return
}
// func dropDisksKeyOnZeroSize(config map[string]interface{}) {
//     // Check if "disks" key exists.
//     if disks, ok := config["disks"].([]interface{}); ok {
//         // Iterate over each disk in the "disks" array.
//         for _, disk := range disks {
//             if d, ok := disk.(map[string]interface{}); ok {
//                 // Check if "size_mib" key exists and is 0.
//                 if size, ok := d["size_mib"].(int); ok && size == 0 {
//                     // If "size_mib" is 0, remove the "disks" key from the config
//                     delete(config, "disks")
//                     break
//                 }
//             }
//         }
//     }
// }
// Delete Qemu VM
func (c *Client) DeleteQemuVm(vmr *VmRef) (err error) {
	url := fmt.Sprintf("/vm/v3/host/%d", vmr.vmId)
        var data map[string]interface{}

        _, err = c.session.DeleteJSON(url, nil, nil, nil, &data)
        if err != nil {
                return
        }
	if data == nil {
		return fmt.Errorf("Can't delete VM %v", vmr.vmId)
	}
        err = c.WaitForCompletion(data)
        return
}
// Delete VMmanager's network
func (c *Client) DeleteNetwork(id string) (err error) {
	url := fmt.Sprintf("/ip/v3/ipnet/%s", id)
        var data map[string]interface{}

        _, err = c.session.DeleteJSON(url, nil, nil, nil, &data)
        if err != nil {
                return
        }
	if data == nil {
		return fmt.Errorf("Can't delete network %v", id)
	}
        return
}
// Update VM's resources
func (c *Client) UpdateQemuResources(vmr *VmRef, config ResourcesQemu) (err error) {
	url := fmt.Sprintf("/vm/v3/host/%d/resource", vmr.vmId)
        var data map[string]interface{}

        _, err = c.session.PostJSON(url, nil, nil, &config, &data)
        if err != nil {
                return
        }
	if data == nil {
		return fmt.Errorf("Can't update VM %v resources", vmr.vmId)
	}
        err = c.WaitForCompletion(data)
        return
}
// Change VM's disk size
func (c *Client) UpdateQemuDisk(config ConfigDisk) (err error) {
	url := fmt.Sprintf("/vm/v3/disk/%d", config.Id)
        var data map[string]interface{}
	size := map[string]int{ "size_mib": config.Size }
        _, err = c.session.PostJSON(url, nil, nil, &size, &data)
        if err != nil {
                return
        }
	if data == nil {
		return fmt.Errorf("Can't update DISK %v size", config.Id)
	}
        err = c.WaitForCompletion(data)
        return
}
// Update configuration of VM
func (c *Client) UpdateQemuConfig(vmr *VmRef, config UpdateConfigQemu) (err error) {
	url := fmt.Sprintf("/vm/v3/host/%d", vmr.vmId)
        var data map[string]interface{}

        _, err = c.session.PostJSON(url, nil, nil, &config, &data)
        if err != nil {
                return
        }
	if data == nil {
		return fmt.Errorf("Can't update VM %v config", vmr.vmId)
	}
        return
}
// Reinstall VM to new OS
func (c *Client) ReinstallQemu(vmr *VmRef, config ReinstallOS) (err error) {
	url := fmt.Sprintf("/vm/v3/host/%d/reinstall", vmr.vmId)
        var data map[string]interface{}

        _, err = c.session.PostJSON(url, nil, nil, &config, &data)
        if err != nil {
                return
        }
	if data == nil {
		return fmt.Errorf("Can't reinstall VM %v", vmr.vmId)
	}
        err = c.WaitForCompletion(data)
        return
}
// Change password of VM
func (c *Client) ChangePassword(vmr *VmRef, password string) (err error) {
	url := fmt.Sprintf("/vm/v3/host/%d/password", vmr.vmId)
        var data map[string]interface{}
	config := map[string]string{"password": password}

        _, err = c.session.PostJSON(url, nil, nil, &config, &data)
        if err != nil {
                return
        }
	if data == nil {
		return fmt.Errorf("Can't change VM %v password", vmr.vmId)
	}
        err = c.WaitForCompletion(data)
        return
}
// Change owner of VM
func (c *Client) ChangeOwner(vmr *VmRef, owner int) (err error) {
	url := fmt.Sprintf("/vm/v3/host/%d/account", vmr.vmId)
        var data map[string]interface{}
	config := map[string]int{"account": owner}
        _, err = c.session.PostJSON(url, nil, nil, &config, &data)
        if err != nil {
                return
        }
	if data == nil {
		return fmt.Errorf("Can't change VM %v owner", vmr.vmId)
	}
        err = c.WaitForCompletion(data)
        return
}
// Get exit status for task in VMmanager
func (c *Client) GetTaskExitstatus(taskUpid int) (exitStatus string, err error) {
        url := fmt.Sprintf("/vm/v3/task?where=consul_id+EQ+%v", taskUpid)
        var data map[string]interface{}
        _, err = c.session.GetJSON(url, nil, nil, &data)
        if err == nil {
		tasks := data["list"].([]interface{})
		task := tasks[0].(map[string]interface{})
                exitStatus = task["status"].(string)
        }
        if exitStatus != exitStatusSuccess {
                err = fmt.Errorf(exitStatus)
        }
        return
}
// WaitForCompletion - poll the API for task completion
func (c *Client) WaitForCompletion(taskResponse map[string]interface{}) (err error) {
        if taskResponse["error"] != nil {
                errJSON, _ := json.MarshalIndent(taskResponse["error"], "", "  ")
		return fmt.Errorf("error reponse: %v", string(errJSON))
        }
        if taskResponse["task"] == nil {
                return nil
        }
        waited := 0
        taskUpid := int(taskResponse["task"].(float64))
        for waited < c.TaskTimeout {
                _, statErr := c.GetTaskExitstatus(taskUpid)
                if statErr == nil {
                        return nil
                }
                time.Sleep(TaskStatusCheckInterval * time.Second)
                waited = waited + TaskStatusCheckInterval
        }
        return fmt.Errorf("Wait timeout for: %v", taskUpid)
}
// Create new network in VMmanager
func (c *Client) CreateNetwork(netParams ConfigNewNetwork) (vmid string, err error) {
        var data map[string]interface{}
        _, err = c.session.PostJSON("/vm/v3/userspace/public/ipnet", nil, nil, &netParams, &data)
        if err != nil {
                return "", err
        }
	if data == nil {
		return "", fmt.Errorf("Can't create network with params %v", netParams)
	}
	vmid = fmt.Sprint(data["id"].(float64))
        return
}
// Get information abount network
func (c *Client) GetNetworkInfo(id string) (netInfo map[string]interface{}, err error) {
	var netlist map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/ip/v3/ipnet?where=id+EQ+%v", id), &netlist, 3)
	if err != nil {
		return nil, err
	}
	if len(netlist["list"].([]interface{})) == 0 {
		return nil, fmt.Errorf("can't find network id %v", id)
	}
	nets := netlist["list"].([]interface{})
	netInfo = nets[0].(map[string]interface{})
        return
}
// Get IP array for VM
func (c *Client) GetVmIpsInfo(vmr *VmRef) (ips []interface{}, err error) {
	var iplist map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/vm/v3/host/%d/ipv4", vmr.vmId), &iplist, 3)
	if err != nil {
		return nil, err
	}
	if len(iplist["list"].([]interface{})) == 0 {
		return nil, fmt.Errorf("can't find ips for vm %v", vmr.vmId)
	}
	ips = iplist["list"].([]interface{})
        return
}
// Update PTR domain record for given IP's id
func (c *Client) UpdatePtr(id int, domain string) (err error) {
	params := map[string]string {
		"domain": domain,
	}
        _, err = c.session.PostJSON(fmt.Sprintf("/vm/v3/ip/%d/ptr", id), nil, nil, &params, nil)
	return
}
// Create pool of IPs in VMmanager
func (c *Client) CreatePool(config ConfigNewPool) (vmid string, err error) {
        var data map[string]interface{}
	// 1. Create pool
	poolParams := map[string]string{
		"name": config.Name,
		"note": config.Note,
        }
        _, err = c.session.PostJSON("/ip/v3/userspace/public/ippool", nil, nil, &poolParams, &data)
        if err != nil {
                return "", err
        }
	if data == nil {
		return "", fmt.Errorf("Can't create Pool with params %v", poolParams)
	}
	vmid = fmt.Sprint(data["id"].(float64))
	// 2. Add ranges to pool
	for _, Range := range config.Ranges {
		err = c.CreatePoolRange(vmid, Range)
		if err != nil {
	                return "", err
	        }
	}
	// 3. Apply pool to cluster
	poolCluster := map[string][]map[string]int{
		"clusters": {
		{
			"id": config.Cluster,
			"interface": 0,
		},
	},
	}
        _, err = c.session.PostJSON(fmt.Sprintf("/vm/v3/ippool/%s/cluster", vmid), nil, nil, &poolCluster, nil)
        if err != nil {
                return "", err
        }
        return
}
// Create account in VMmanager
func (c *Client) CreateAccount(config ConfigNewAccount) (vmid string, err error) {
        var data map[string]interface{}
        _, err = c.session.PostJSON("/vm/v3/account", nil, nil, &config, &data)
        if err != nil {
                return "", err
        }
	if data == nil {
		return "", fmt.Errorf("Can't create Account with params %v", config)
	}
	vmid = fmt.Sprint(data["id"].(float64))

        return
}
// Update setting for pool
func (c *Client) UpdatePoolSettings(poolId string, name string, desc string) (err error) {
	rangeObject := map[string]string {
		"name": name,
		"note": desc,
	}
        _, err = c.session.PostJSON(fmt.Sprintf("/ip/v3/ippool/%s", poolId), nil, nil, &rangeObject, nil)
	return
}
// Update network description
func (c *Client) UpdateNetworkDescription(id string, desc string) (err error) {
	rangeObject := map[string]string {
		"note": desc,
	}
        _, err = c.session.PostJSON(fmt.Sprintf("/vm/v3/ipnet/%s", id), nil, nil, &rangeObject, nil)
	return
}
// Create new IPs range in Pool
func (c *Client) CreatePoolRange(poolId string, rangestring string) (err error) {
        var data map[string]interface{}
        rangeObject := map[string]string {
		"name": rangestring,
        }
        _, err = c.session.PostJSON(fmt.Sprintf("/vm/v3/ippool/%s/range", poolId), nil, nil, &rangeObject, &data)
        if err != nil {
                return err
        }
	if data == nil {
		return fmt.Errorf("Can't create Pool with params %v", rangeObject)
	}
	return
}
// Delete range from pool
func (c *Client) DeletePoolRange(rangeId int) (err error) {
        _, err = c.session.DeleteJSON(fmt.Sprintf("/ip/v3/range/%d", rangeId), nil, nil, nil, nil)
        if err != nil {
                return err
        }
	return
}
// Get information about Pool
func (c *Client) GetPoolInfo(id string) (config map[string]interface{}, err error) {
	var poolinfo map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/ip/v3/ippool/%v", id), &poolinfo, 3)
	if err != nil {
		return nil, err
	}
	var configPool ConfigPool
	configPool.Id = id
	configPool.Name = poolinfo["name"].(string)
	configPool.Note = poolinfo["note"].(string)
	var ranges map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/vm/v3/range?where=ippool+EQ+%v", id), &ranges, 3)
	if err != nil {
		return nil, err
	}
	if ranges["list"] == nil {
		return nil, fmt.Errorf("can't find Ranges in pool id %v", id)
	}
	for _, Range := range ranges["list"].([]interface{}) {
		RangeObject := Range.(map[string]interface{})
		var NewRange ConfigPoolRanges
		NewRange.Range = RangeObject["name"].(string)
		NewRange.Id = int(RangeObject["id"].(float64))
		configPool.Ranges = append(configPool.Ranges, NewRange)
	}
	j, err := json.Marshal(configPool)
	err = json.Unmarshal(j, &config)
	return
}
// Get information about Account
func (c *Client) GetAccountInfo(id string) (config map[string]interface{}, err error) {
	var data map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/vm/v3/account?where=id+EQ+%v", id), &data, 3)
	if err != nil {
		return nil, err
	}
	var account ConfigAccount
	if len(data["list"].([]interface{})) == 0 {
		return nil, fmt.Errorf("can't find user with id %v", id)
	}
	foundAcc := data["list"].([]interface{})[0].(map[string]interface{})
	account.Id = int(foundAcc["id"].(float64))
	account.State = foundAcc["state"].(string)
	account.Role = foundAcc["roles"].([]interface{})[0].(string)
	account.Email = foundAcc["email"].(string)

	j, err := json.Marshal(account)
	err = json.Unmarshal(j, &config)
	return
}
// Find pool by name
func (c *Client) GetPoolIdByName(name string) (id string, err error) {
	var poolinfo map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/ip/v3/ippool?where=name+CP+%%27%s%%27", name), &poolinfo, 3)
	if err != nil {
		return "", err
	}
	if len(poolinfo["list"].([]interface{})) == 0 {
		return "0", nil
	}
	id = fmt.Sprint(poolinfo["list"].([]interface{})[0].(map[string]interface{})["id"].(float64))
	return
}
// Find network by name
func (c *Client) GetNetworkIdByName(name string) (id string, err error) {
	var poolinfo map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/ip/v3/ipnet?where=name+CP+%%27%s%%27", name), &poolinfo, 3)
	if err != nil {
		return "", err
	}
	if len(poolinfo["list"].([]interface{})) == 0 {
		return "0", nil
	}
	id = fmt.Sprint(poolinfo["list"].([]interface{})[0].(map[string]interface{})["id"].(float64))
	return
}
// Find account by email
func (c *Client) GetAccountIdByEmail(email string) (id string, err error) {
	var data map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/vm/v3/account?where=email+CP+%%27%s%%27", email), &data, 3)
	if err != nil {
		return "", err
	}
	if len(data["list"].([]interface{})) == 0 {
		return "0", nil
	}
	id = fmt.Sprint(data["list"].([]interface{})[0].(map[string]interface{})["id"].(float64))
	return
}
// Delete IPs pool
func (c *Client) DeletePool(id string) (err error) {
	url := fmt.Sprintf("/ip/v3/ippool/%s", id)
        var data map[string]interface{}

        _, err = c.session.DeleteJSON(url, nil, nil, nil, &data)
        if err != nil {
                return
        }
	if data == nil {
		return fmt.Errorf("Can't delete Pool %v", id)
	}
        return
}
// Delete account
func (c *Client) DeleteAccount(id string) (err error) {
	url := fmt.Sprintf("/vm/v3/user/%s", id)
        var data map[string]interface{}

        _, err = c.session.DeleteJSON(url, nil, nil, nil, &data)
        if err != nil {
                return
        }
	if data == nil {
		return fmt.Errorf("Can't delete account %v", id)
	}
        return
}
// Change account role
func (c *Client) ChangeAccountRole(id string, role string) (err error) {
	url := fmt.Sprintf("/vm/v3/user/%v", id)
	config := map[string][]string{"roles": { role }}
        _, err = c.session.PostJSON(url, nil, nil, &config, nil)
	return
}
// Add ssh public key to account
func (c *Client) AccountAddSshKey(id string, key SshKeyConfig) (err error) {
	url := fmt.Sprintf("/auth/v3/user/%v/sshkey", id)
	config := map[string]string{
		"name": key.Name,
		"ssh_pub_key": key.Key,
	}
        _, err = c.session.PostJSON(url, nil, nil, &config, nil)
	return
}
// Get set of ssh public keys from account 
func (c *Client) AccountGetSshKeys(id string) (ssh_keys []interface{}, err error) {
	var data map[string]interface{}
	url := fmt.Sprintf("/auth/v3/user/%v/sshkey", id)
        err = c.GetJsonRetryable(url, &data, 3)
	if err != nil {
		return nil, err
	}
	if len(data["list"].([]interface{})) == 0 {
		return nil, nil
	}
	ssh_keys = data["list"].([]interface{})
	return
}
// Add VxLAN network to some account
func (c *Client) AccountAddVxLAN(config ConfigNewVxLAN) (id string, err error) {
	var data map[string]interface{}
        _, err = c.session.PostJSON("/vm/v3/vxlan", nil, nil, &config, &data)
	if err != nil {
                return "", err
        }
	if data == nil {
		return "", fmt.Errorf("Can't create VxLAN with params %#v", config)
	}
	id = fmt.Sprint(data["id"].(float64))
	return
}
// Get information about VxLAN
func (c *Client) GetVxLANInfo(id string) (config map[string]interface{}, err error) {
	var data map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/vm/v3/vxlan?where=id+EQ+%v", id), &data, 3)
	if err != nil {
		return nil, err
	}
	if len(data["list"].([]interface{})) == 0 {
		return nil, fmt.Errorf("can't find VxLAN id %v", id)
	}
	config = data["list"].([]interface{})[0].(map[string]interface{})
	return
}
// Delete VxLAN
func (c *Client) DeleteVxLAN(id string) (err error) {
	url := fmt.Sprintf("/vm/v3/vxlan/%s", id)
        var data map[string]interface{}

        _, err = c.session.DeleteJSON(url, nil, nil, nil, &data)
        if err != nil {
                return
        }
	if data == nil {
		return fmt.Errorf("Can't delete VxLAN %v", id)
	}
        return
}
// Find VxLAN by name and account
func (c *Client) GetVxLANIdByName(account int, name string) (id string, err error) {
	var data map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/vm/v3/vxlan?where=(name+CP+%%27%s%%27)+AND+(account.id+EQ+%d)", name, account), &data, 3)
	if err != nil {
		return "", err
	}
	if len(data["list"].([]interface{})) == 0 {
		return "0", nil
	}
	id = fmt.Sprint(data["list"].([]interface{})[0].(map[string]interface{})["id"].(float64))
	return
}
