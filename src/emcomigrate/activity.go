// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package emcomigrate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// TODO REVISIT Copied from EMCO as import leads to conflicts
type GenericPlacementIntent struct {
	MetaData GenIntentMetaData `json:"metadata"`
}

// GenIntentMetaData has name, description, userdata1, userdata2
type GenIntentMetaData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UserData1   string `json:"userData1"`
	UserData2   string `json:"userData2"`
}

// AppIntent has two components - metadata, spec
type AppIntent struct {
	MetaData MetaData `json:"metadata,omitempty"`
	Spec     SpecData `json:"spec,omitempty"`
}

// MetaData has - name, description, userdata1, userdata2
type MetaData struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	UserData1   string `json:"userData1,omitempty"`
	UserData2   string `json:"userData2,omitempty"`
}

// SpecData consists of appName and intent
type SpecData struct {
	AppName string      `json:"app,omitempty"`
	Intent  IntentStruc `json:"intent,omitempty"`
}

type IntentStruc struct {
	AllOfArray []AllOf `json:"allOf,omitempty"`
	AnyOfArray []AnyOf `json:"anyOf,omitempty"`
}

// AllOf consists of ProviderName, ClusterName, ClusterLabelName and AnyOfArray. Any of   them can be empty
type AllOf struct {
	ProviderName     string  `json:"clusterProvider,omitempty"`
	ClusterName      string  `json:"cluster,omitempty"`
	ClusterLabelName string  `json:"clusterLabel,omitempty"`
	AnyOfArray       []AnyOf `json:"anyOf,omitempty"`
}

// AnyOf consists of Array of ProviderName & ClusterLabelNames
type AnyOf struct {
	ProviderName     string `json:"clusterProvider,omitempty"`
	ClusterName      string `json:"cluster,omitempty"`
	ClusterLabelName string `json:"clusterLabel,omitempty"`
}

// GetDigAppIntents gets all the app intents for the given Deployment Intent Group.
// A DIG has one or more Generic Placement Intents (GPI) and each GPI has one or
// more app intents. An app intent specifies the cluster mapping for a
// single app (helm chart).
func GetDigAppIntents(ctx context.Context, migParam MigParam) (*MigParam, error) {

	fmt.Printf("GetDigAppIntents got params: %#v\n", migParam)

	gpiUrl := buildGenericPlacementIntentsURL(migParam.InParams)
	fmt.Printf("\nGetDigAppIntents: gpiUrl = %s\n", gpiUrl)

	respBody, err := getHttpRespBody(gpiUrl)
	if err != nil {
		return nil, err
	}
	migParam.GenericPlacementIntentURL = gpiUrl

	var gpIntents []GenericPlacementIntent
	if err := json.Unmarshal(respBody, &gpIntents); err != nil {
		decodeErr := fmt.Errorf("Failed to decode GET responde body for URL %s.\n"+
			"Decoder error: %#v\n", gpiUrl, err)
		fmt.Fprintf(os.Stderr, decodeErr.Error())
		return nil, decodeErr
	}
	fmt.Printf("\nGetDigAppIntents: body = %#v\n", gpIntents)

	migParam.AppNameIntentPairs = make(map[string][]AppNameIntentPair)

	for _, gpIntent := range gpIntents {
		appIntentsUrl := buildAppIntentsURL(gpiUrl, gpIntent.MetaData.Name)

		respBody, err := getHttpRespBody(appIntentsUrl)
		if err != nil {
			return nil, err
		}

		var appIntents []AppIntent
		if err := json.Unmarshal(respBody, &appIntents); err != nil {
			decodeErr := fmt.Errorf("Failed to decode GET responde body for "+
				"URL %s.\nDecoder error: %#v\n", appIntentsUrl, err)
			fmt.Fprintf(os.Stderr, decodeErr.Error())
			return nil, decodeErr
		}
		fmt.Printf("\nGetDigAppIntents: body = %#v\n", appIntents)

		// Build list of appName/appIbtentName pairs for this gpIntent
		appIntentNames := make([]AppNameIntentPair, 0, len(appIntents))
		for _, appIntent := range appIntents {
			pair := AppNameIntentPair{
				AppName:       appIntent.Spec.AppName,
				AppIntentName: appIntent.MetaData.Name,
			}
			appIntentNames = append(appIntentNames, pair)
		}
		migParam.AppNameIntentPairs[gpIntent.MetaData.Name] = appIntentNames
	}

	return &migParam, nil
}

// UpdateAppIntents updates the app intents for a DIG to map all apps in that
// DIG to a given target cluster. It builds the modified app intents locally
// and then does a POST call to EMCO API to update the DIG's app intents.
// The actual app migration happens only in the next activity, not here.
func UpdateAppIntents(ctx context.Context, migParam MigParam) (*MigParam, error) {

	// Update the intents, walking through migParam.AppsNameDetails map
	newAppSpecIntent := IntentStruc{ // all apps get this spec intent
		AllOfArray: []AllOf{
			{
				ProviderName: migParam.InParams["targetClusterProvider"],
				ClusterName:  migParam.InParams["targetClusterName"],
			},
		},
	}

	for gpIntentName, appNameIntentPairs := range migParam.AppNameIntentPairs {
		appIntentBaseURL := buildAppIntentsURL(
			migParam.GenericPlacementIntentURL, gpIntentName)
		for _, appNameIntentPair := range appNameIntentPairs {
			appIntentURL := appIntentBaseURL + "/" + appNameIntentPair.AppIntentName
			newAppIntent := AppIntent{
				MetaData: MetaData{Name: appNameIntentPair.AppIntentName},
				Spec: SpecData{
					AppName: appNameIntentPair.AppName,
					Intent:  newAppSpecIntent,
				},
			}
			appIntentJSON, err := json.Marshal(newAppIntent)
			if err != nil {
				encodeErr := fmt.Errorf("Error marshaling appIntent %#v\n"+
					"Marshal error; %#v\n", newAppIntent, err)
				fmt.Fprintf(os.Stderr, encodeErr.Error())
				return nil, encodeErr
			}

			fmt.Printf("\nappIntentURL: %s\nappIntent: %#v\n\n",
				appIntentURL, newAppIntent)

			req, err := http.NewRequest(http.MethodPut,
				appIntentURL, bytes.NewBuffer(appIntentJSON))
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				putErr := fmt.Errorf("HTTP PUT failed for URL %s.\nError: %s\n",
					appIntentURL, err)
				fmt.Fprintf(os.Stderr, putErr.Error())
				return nil, putErr
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				putErr := fmt.Errorf("HTTP PUT returned status code %s for URL %s.\n",
					resp.Status, appIntentURL)
				fmt.Fprintf(os.Stderr, putErr.Error())
				return nil, putErr
			}
		}
	}

	return &migParam, nil
}

// DoDigUpdate calls EMCO's /update API to migrate the app.
func DoDigUpdate(ctx context.Context, migParam MigParam) (*MigParam, error) {

	// POST dig update operation
	digURL := buildDigURL(migParam.InParams)
	digUpdateURL := digURL + "/update"
	resp, err := http.Post(digUpdateURL, "", nil)
	if err != nil {
		postErr := fmt.Errorf("HTTP POST failed for URL %s.\nError: %s\n",
			digUpdateURL, err)
		fmt.Fprintf(os.Stderr, postErr.Error())
		return nil, postErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		postErr := fmt.Errorf("HTTP POST returned status code %s for URL %s.\n",
			resp.Status, digUpdateURL)
		fmt.Fprintf(os.Stderr, postErr.Error())
		return nil, postErr
	}

	return &migParam, nil
}

func buildDigURL(params map[string]string) string {
	url := params["emcoURL"]
	url += "/v2/projects/" + params["project"]
	url += "/composite-apps/" + params["compositeApp"]
	url += "/" + params["compositeAppVersion"]
	url += "/deployment-intent-groups/" + params["deploymentIntentGroup"]

	return url
}

func buildGenericPlacementIntentsURL(params map[string]string) string {
	url := buildDigURL(params)
	url += "/generic-placement-intents"

	return url
}

func buildAppIntentsURL(gpiURL string, gpiName string) string {
	url := gpiURL + "/" + gpiName + "/app-intents"
	return url
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
