// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package nvidiawf

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func buildDigURL(params map[string]string) string {
	url := params["emcoURL"]
	url += "/v2/projects/" + params["project"]
	url += "/composite-apps/" + params["compositeApp"]
	url += "/" + params["compositeAppVersion"]
	url += "/deployment-intent-groups/" + params["deploymentIntentGroup"]

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

// DoDigApprove calls EMCO's /instantiate API
func DoDigApprove(ctx context.Context, params NwfParam) (*NwfParam, error) {

	// POST dig update operation
	fmt.Printf("Approve XXXXXXXXX: migParam = %#v\n", params.InParams)
	digURL := buildDigURL(params.InParams)
	digInstantiateURL := digURL + "/approve"
	resp, err := http.Post(digInstantiateURL, "", nil)
	if err != nil {
		postErr := fmt.Errorf("HTTP POST failed for URL %s.\nError: %s\n",
			digInstantiateURL, err)
		fmt.Fprintf(os.Stderr, postErr.Error())
		return nil, postErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		postErr := fmt.Errorf("HTTP POST returned status code %s for URL %s.\n",
			resp.Status, digInstantiateURL)
		fmt.Fprintf(os.Stderr, postErr.Error())
		return nil, postErr
	}

	return &params, nil
}

// DoDigInstantiate calls EMCO's /instantiate API
func DoDigInstantiate(ctx context.Context, params NwfParam) (*NwfParam, error) {

	// POST dig update operation
	fmt.Printf("XXXXXXXXX: migParam = %#v\n", params.InParams)
	digURL := buildDigURL(params.InParams)
	digInstantiateURL := digURL + "/instantiate"
	resp, err := http.Post(digInstantiateURL, "", nil)
	if err != nil {
		postErr := fmt.Errorf("HTTP POST failed for URL %s.\nError: %s\n",
			digInstantiateURL, err)
		fmt.Fprintf(os.Stderr, postErr.Error())
		return nil, postErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		postErr := fmt.Errorf("HTTP POST returned status code %s for URL %s.\n",
			resp.Status, digInstantiateURL)
		fmt.Fprintf(os.Stderr, postErr.Error())
		return nil, postErr
	}

	return &params, nil
}

// DoDigTerminate calls EMCO's /terminate API
func DoDigTerminate(ctx context.Context, params NwfParam) (*NwfParam, error) {

	// POST dig update operation
	digURL := buildDig1URL(params.InParams)
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

// func getHttpRespBody(url string) (io.ReadCloser, error) {
func getHttpRespBody(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		getErr := fmt.Errorf("HTTP GET failed for URL %s.\nError: %s\n",
			url, err)
		fmt.Fprintf(os.Stderr, getErr.Error())
		return nil, getErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		getErr := fmt.Errorf("HTTP GET returned status code %s for URL %s.\n",
			resp.Status, url)
		fmt.Fprintf(os.Stderr, getErr.Error())
		return nil, getErr
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	return b, nil
}
