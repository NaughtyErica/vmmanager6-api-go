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

func NewClient(apiUrl string, hclient *http.Client, tls *tls.Config, taskTimeout int) (client *Client, err error) {
	var sess *Session
	sess, err = NewSession(apiUrl, hclient, tls)
	if err == nil {
		client = &Client{session: sess, ApiUrl: apiUrl, TaskTimeout: taskTimeout}
	}
	return client, err
}
func (c *Client) SetAPIToken(token string) {
	c.session.SetAPIToken(token)
}
func (c *Client) Login(username string, password string) (err error) {
	c.Username = username
	c.Password = password
	return c.session.Login(username, password)
}
func (c *Client) GetItemConfigMapStringInterface(url, text string) (map[string]interface{}, error) {
	data, err := c.GetItemConfig(url, text)
	if err != nil {return nil, err}
	return data["data"].(map[string]interface{}), err
}

func (c *Client) GetItemConfigString(url, text string) (string, error) {
	data, err := c.GetItemConfig(url, text)
	if err != nil {return "", err}
	return data["data"].(string), err
}

func (c *Client) GetItemConfigInterfaceArray(url, text string) ([]interface{}, error) {
	data, err := c.GetItemConfig(url, text)
	if err != nil {return nil, err}
	return data["data"].([]interface{}), err
}

func (c *Client) GetItemConfig(url, text string) (config map[string]interface{}, err error) {
	err = c.GetJsonRetryable(url, &config, 3)
	if err != nil {return nil, err}
	if config["data"] == nil {return nil, fmt.Errorf(text + " CONFIG not readable")}
	return
}

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
func (c *Client) GetNodeList() (list map[string]interface{}, err error) {
	err = c.GetJsonRetryable("/node", &list, 3)
	return
}
func (c *Client) GetVmList() (list map[string]interface{}, err error) {
	err = c.GetJsonRetryable("/host", &list, 3)
	return
}
func (c *Client) DeleteUrl(url string) (err error) {
	_, err = c.session.Delete(url, nil, nil)
	return
}
func NewVmRef(vmId int) (vmr *VmRef) {
        vmr = &VmRef{vmId: vmId}
        return
}

func (c *Client) GetVmInfo(vmr *VmRef) (vmInfo map[string]interface{}, err error) {
	var vmlist map[string]interface{}
	err = c.GetJsonRetryable(fmt.Sprintf("/host?where=id+EQ+%v", vmr.vmId), &vmlist, 3)
	if err != nil {
		return nil, err
	}
	if vmlist["list"] == nil {
		return nil, fmt.Errorf("can't find vm id %v", vmr.vmId)
	}
	vms := vmlist["list"].([]interface{})
	vmInfo = vms[0].(map[string]interface{})
        return
}

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

func (c *Client) CreateQemuVm(vmParams ConfigNewQemu) (vmid int, err error) {
        var data map[string]interface{}
        _, err = c.session.PostJSON("/host", nil, nil, &vmParams, &data)
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

func (c *Client) DeleteQemuVm(vmr *VmRef) (err error) {
	url := fmt.Sprintf("/host/%d", vmr.vmId)
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
func (c *Client) UpdateQemuResources(vmr *VmRef, config ResourcesQemu) (err error) {
	url := fmt.Sprintf("/host/%d/resource", vmr.vmId)
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
func (c *Client) UpdateQemuDisk(config ConfigDisk) (err error) {
	url := fmt.Sprintf("/disk/%d", config.Id)
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

func (c *Client) UpdateQemuConfig(vmr *VmRef, config UpdateConfigQemu) (err error) {
	url := fmt.Sprintf("/host/%d", vmr.vmId)
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

func (c *Client) ReinstallQemu(vmr *VmRef, config ReinstallOS) (err error) {
	url := fmt.Sprintf("/host/%d/reinstall", vmr.vmId)
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

func (c *Client) ChangePassword(vmr *VmRef, password string) (err error) {
	url := fmt.Sprintf("/host/%d/password", vmr.vmId)
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

func (c *Client) ChangeOwner(vmr *VmRef, owner int) (err error) {
	url := fmt.Sprintf("/host/%d/account", vmr.vmId)
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

func (c *Client) GetTaskExitstatus(taskUpid int) (exitStatus string, err error) {
        url := fmt.Sprintf("/task?where=consul_id+EQ+%v", taskUpid)
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
