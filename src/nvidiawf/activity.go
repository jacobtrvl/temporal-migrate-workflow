// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package nvidiawf

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type digActions struct {
	State    string    `json:"state"`
	Instance string    `json:"instance"`
	Time     time.Time `json:"time"`
	Revision int       `json:"revision"`
}
type AppsStatus struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Clusters    []struct {
		ClusterProvider string `json:"clusterProvider"`
		Cluster         string `json:"cluster"`
		Connectivity    string `json:"connectivity"`
		Resources       []struct {
			GVK struct {
				Group   string `json:"Group"`
				Version string `json:"Version"`
				Kind    string `json:"Kind"`
			} `json:"GVK"`
			Name string `json:"name"`
			//RsyncStatus    string `json:"rsyncStatus,omitempty"`
			//ClusterStatus  string `json:"clusterStatus,omitempty"`
			DeployedStatus string `json:"deployedStatus"`
			ReadyStatus    string `json:"readyStatus"`
		} `json:"resources"`
	} `json:"clusters"`
}

type digStatus struct {
	Project              string `json:"project"`
	CompositeAppName     string `json:"compositeApp"`
	CompositeAppVersion  string `json:"compositeAppVersion"`
	CompositeProfileName string `json:"compositeProfile"`
	Name                 string `json:"name"`
	States               struct {
		Actions []digActions `json:"actions"`
	} `json:"states"`
	ReadyStatus    string `json:"readyStatus,omitempty"`
	DeployedStatus string `json:"deployedStatus"`
	//RsyncStatus    struct {
	// Deleted int `json:"Deleted,omitempty"`
	//} `json:"rsyncStatus,omitempty"`
	DeployedCounts struct {
		Applied int `json:"Applied"`
	} `json:"deployedCounts"`
	//ClusterStatus struct {
	// NotReady int `json:"NotReady,omitempty"`
	// Ready    int `json:"Ready,omitempty"`
	//} `json:"clusterStatus,omitempty"`
	Apps          []AppsStatus `json:"apps,omitempty"`
	IsCheckedOut  bool         `json:"isCheckedOut"`
	TargetVersion string       `json:"targetVersion"`
	DigId         string       `json:"digId,omitempty"`
}

func buildDigURL(params map[string]string) string {
	url := params["emcoURL"]
	url += "/v2/projects/" + params["project"]
	url += "/composite-apps/" + params["compositeApp"]
	url += "/" + params["compositeAppVersion"]
	url += "/deployment-intent-groups/" + params["deploymentIntentGroup"]

	return url
}

func buildMiddleendURL(params map[string]string) string {
	url := params["middleendURL"]
	url += "/middleend/projects/" + params["project"]
	url += "/composite-apps/" + params["compositeApp1"]
	url += "/" + params["compositeAppVersion1"]
	url += "/deployment-intent-groups/" + params["deploymentIntentGroup1"]

	return url
}
func buildMiddleendMECURL(params map[string]string) string {
	url := params["middleendURL"]
	url += "/middleend/projects/" + params["project"]
	url += "/composite-apps/" + params["mecApp"]
	url += "/" + params["mecAppVersion"]
	url += "/deployment-intent-groups/" + params["mecAppDig"]

	return url
}

func buildDig1URL(params map[string]string) string {
	url := params["emcoURL"]
	url += "/v2/projects/" + params["project"]
	url += "/composite-apps/" + params["compositeApp1"]
	url += "/" + params["compositeAppVersion1"]
	url += "/deployment-intent-groups/" + params["deploymentIntentGroup1"]

	return url
}

func buildMECDigURL(params map[string]string) string {
	url := params["emcoURL"]
	url += "/v2/projects/" + params["project"]
	url += "/composite-apps/" + params["mecApp"]
	url += "/" + params["mecAppVersion"]
	url += "/deployment-intent-groups/" + params["mecAppDig"]

	return url
}

func getDigStatus(middleendURL string, statusType string) (string, error) {
	resp, err := http.Get(middleendURL)
	if err != nil {
		postErr := fmt.Errorf("HTTP POST failed for URL %s.\nError: %s\n",
			middleendURL, err)
		fmt.Fprintf(os.Stderr, postErr.Error())
		return "", postErr
	}

	if resp.StatusCode != http.StatusOK {
		getErr := fmt.Errorf("HTTP GET returned status code %s for URL %s.\n",
			resp.Status, middleendURL)
		fmt.Fprintf(os.Stderr, getErr.Error())
		return "", getErr
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	status := digStatus{}
	err = json.Unmarshal(b, &status)
	if err != nil {
		Err := fmt.Errorf("Failedto unmarashal.\nError: %s\n", err)
		fmt.Fprintf(os.Stderr, Err.Error())
	}

	if statusType == "deployed" {
		return status.DeployedStatus, nil
	} else {
		return status.ReadyStatus, nil
	}
}

// DoDigApprove calls EMCO's /instantiate API
func DoDigApprove(ctx context.Context, params NwfParam) (*NwfParam, error) {
	var digURL = ""
	var middleendURL = ""

	// POST dig update operation
	fmt.Printf("Approve XXXXXXXXX: migParam = %#v\n", params.InParams)

	if params.App == "MEC" {
		digURL = buildMECDigURL(params.InParams)
		middleendURL = buildMiddleendMECURL(params.InParams) + "/status"
	} else {
		digURL = buildDig1URL(params.InParams)
		middleendURL = buildMiddleendURL(params.InParams) + "/status"
	}

	// Get the status of the DIG
	params.mu.Lock()
	status, err := getDigStatus(middleendURL, "deployed")
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		params.mu.Unlock()
		return nil, err
	}
	if status == "Approved" {
		fmt.Printf("DIG in Approved state already: %s", middleendURL)
		params.mu.Unlock()
		return &params, nil
	}

	digInstantiateURL := digURL + "/approve"
	resp, err := http.Post(digInstantiateURL, "", nil)
	if err != nil {
		postErr := fmt.Errorf("HTTP POST failed for URL %s.\nError: %s\n",
			digInstantiateURL, err)
		fmt.Fprintf(os.Stderr, postErr.Error())
		params.mu.Unlock()
		return nil, postErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		postErr := fmt.Errorf("HTTP POST returned status code %s for URL %s.\n",
			resp.Status, digInstantiateURL)
		fmt.Fprintf(os.Stderr, postErr.Error())
		params.mu.Unlock()
		return nil, postErr
	}

	params.mu.Unlock()
	return &params, nil
}

// DoDigInstantiate calls EMCO's /instantiate API
func DoDigInstantiate(ctx context.Context, params NwfParam) (*NwfParam, error) {
	var digURL = ""
	var middleendURL = ""

	// POST dig update operation
	fmt.Printf("XXXXXXXXX: migParam = %#v\n", params.InParams)

	if params.App == "MEC" {
		digURL = buildMECDigURL(params.InParams)
		middleendURL = buildMiddleendMECURL(params.InParams) + "/status"
	} else {
		digURL = buildDig1URL(params.InParams)
		middleendURL = buildMiddleendURL(params.InParams) + "/status"
	}

	// Get the status of the DIG
	params.mu.Lock()
	status, err := getDigStatus(middleendURL, "deployed")
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		params.mu.Unlock()
		return nil, err
	}
	if status == "Instantiated" {
		fmt.Printf("DIG in Instantiated state already: %s", middleendURL)
		params.mu.Unlock()
		return &params, nil
	}

	digInstantiateURL := digURL + "/instantiate"
	resp, err := http.Post(digInstantiateURL, "", nil)
	if err != nil {
		postErr := fmt.Errorf("HTTP POST failed for URL %s.\nError: %s\n",
			digInstantiateURL, err)
		fmt.Fprintf(os.Stderr, postErr.Error())
		params.mu.Unlock()
		return nil, postErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		postErr := fmt.Errorf("HTTP POST returned status code %s for URL %s.\n",
			resp.Status, digInstantiateURL)
		fmt.Fprintf(os.Stderr, postErr.Error())
		params.mu.Unlock()
		return nil, postErr
	}

	params.mu.Unlock()
	return &params, nil
}

// DoDigTerminate calls EMCO's /terminate API
func DoDigTerminate(ctx context.Context, params NwfParam) (*NwfParam, error) {

	// POST dig update operation
	digURL := buildDigURL(params.InParams)
	digTerminateURL := digURL + "/terminate"
	resp, err := http.Post(digTerminateURL, "", nil)
	if err != nil {
		postErr := fmt.Errorf("HTTP POST failed for URL %s.\nError: %s\n",
			digTerminateURL, err)
		fmt.Fprintf(os.Stderr, postErr.Error())
		return nil, postErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		postErr := fmt.Errorf("HTTP POST returned status code %s for URL %s.\n",
			resp.Status, digTerminateURL)
		fmt.Fprintf(os.Stderr, postErr.Error())
		return nil, postErr
	}
	return &params, nil
}

// DoSwitchConfig does remote switch config
func DoSwitchConfig(ctx context.Context, params NwfParam) (*NwfParam, error) {
	sshClientURL := params.InParams["sshClientURL"] + "/execute"
	resp, err := http.Post(sshClientURL, "", nil)
	if err != nil {
		postErr := fmt.Errorf("HTTP POST failed for URL %s.\nError: %s\n",
			sshClientURL, err)
		fmt.Fprintf(os.Stderr, postErr.Error())
		return nil, postErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		postErr := fmt.Errorf("HTTP POST returned status code %s for URL %s.\n",
			resp.Status, sshClientURL)
		fmt.Fprintf(os.Stderr, postErr.Error())
		return nil, postErr
	}
	return &params, nil
}

func GetInstantiateStatus(ctx context.Context, params NwfParam) (*NwfParam, error) {
	middleendURL := buildMiddleendURL(params.InParams) + "/status"
	fmt.Printf("YYYYY : status = %#s\n", middleendURL)
	status, err := getDigStatus(middleendURL, "ready")
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return nil, err
	}

	fmt.Printf("YYYYYXXXXX %s\n", status)
	if status != "Ready" {
		err2 := fmt.Errorf("the DU is still not ready %g", status)
		fmt.Fprintf(os.Stderr, err2.Error())
		return nil, err2
	}
	return &params, nil
}
