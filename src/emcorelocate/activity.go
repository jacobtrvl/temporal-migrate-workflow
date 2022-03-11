// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package emcorelocate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
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

// AllOf consists if ProviderName, ClusterName, ClusterLabelName and AnyOfArray. Any of   them can be empty
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

func GetDigAppIntents(ctx context.Context, migParam MigParam) (*MigParam, error) {

	fmt.Printf("GetDigAppIntents got params: %#v\n", migParam)

	gpiUrl := buildGenericPlacementIntentsURL(migParam.InParams)
	fmt.Printf("\nGetDigAppIntents: gpiUrl = %s\n", gpiUrl)

	// statusAnchor will be used to check deployment status
	statusAnchor := buildStatusAnchor(migParam.InParams)
	fmt.Printf("\nGetDigAppIntents: statusAnchor = %s\n", statusAnchor)

	respBody, err := getHttpRespBody(gpiUrl)
	if err != nil {
		return nil, err
	}
	migParam.GenericPlacementIntentURL = gpiUrl
	migParam.StatusAnchor = statusAnchor

	var gpIntents []GenericPlacementIntent
	if err := json.Unmarshal(respBody, &gpIntents); err != nil {
		decodeErr := fmt.Errorf("Failed to decode GET responde body for URL %s.\n"+
			"Decoder error: %#v\n", gpiUrl, err)
		fmt.Fprintf(os.Stderr, decodeErr.Error())
		return nil, decodeErr
	}
	fmt.Printf("\nGetDigAppIntents: body = %#v\n", gpIntents)

	migParam.AppsNameDetails = make(map[string][]AppNameDetails)

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

		// TODO []AppNameDetails shouldn't be a list? We want to relocate just 1 application.
		// leave [] for now to maintain compatibility with original migrate workflow
		var appIntentNames []AppNameDetails
		for _, appIntent := range appIntents {
			if strings.ToLower(appIntent.Spec.AppName) == strings.ToLower(migParam.InParams["targetAppName"]) {
				targetApp := AppNameDetails{
					AppName:       appIntent.Spec.AppName,
					AppIntentName: appIntent.MetaData.Name,
					Phase:         ApplyPhase,
					PrimaryIntent: appIntent.Spec.Intent,
				}
				appIntentNames = append(appIntentNames, targetApp)
			}
		}
		// TODO: We could use "equal", rather tan "less than". We consider just 1 app.
		// If targetAppName wasn't found in the current deployment, return error
		if len(appIntentNames) < 1 {
			err := fmt.Errorf("error: %v targetAppName not found", migParam.InParams["targetAppName"])
			fmt.Fprintf(os.Stderr, err.Error())
			return nil, err
		}
		migParam.AppsNameDetails[gpIntent.MetaData.Name] = appIntentNames
	}

	return &migParam, nil
}

func UpdateAppIntents(ctx context.Context, migParam MigParam) (*MigParam, error) {

	// newAppSpecIntent is target intent for relocated app. For now we assume, that only
	// cluster name can be used. TODO: Consider clusterLabel
	newAppSpecIntent := IntentStruc{ // all apps get this spec intent
		AllOfArray: []AllOf{
			{
				ProviderName: migParam.InParams["targetClusterProvider"],
				ClusterName:  migParam.InParams["targetClusterName"],
			},
		},
	}

	for gpIntentName, appNameDetails := range migParam.AppsNameDetails {
		appIntentBaseURL := buildAppIntentsURL(migParam.GenericPlacementIntentURL, gpIntentName)
		for index, appNameDetails := range appNameDetails {
			switch appNameDetails.Phase {
			case ApplyPhase:
				// For each PrimaryIntent in AllOfArray check if Intent is in the NewPlacementIntent.
				// If not present, append to AllOfArray, to assure service continuity.
				for _, plcIntent := range appNameDetails.PrimaryIntent.AllOfArray {
					skip := checkIfSkipPrimaryIntentAllOf(migParam, plcIntent, newAppSpecIntent.AllOfArray)
					if !skip {
						newAppSpecIntent.AllOfArray = append(newAppSpecIntent.AllOfArray, plcIntent)
					}
				}
				// For each PrimaryIntent in AnyOfArray check if Intent is in the NewPlacementIntent.
				// If not present, append to new AnyOfArray, to assure service continuity.
				for _, plcIntent := range appNameDetails.PrimaryIntent.AnyOfArray {
					skip := checkIfSkipPrimaryIntentAnyOf(migParam, plcIntent, newAppSpecIntent.AnyOfArray)
					if !skip {
						newAppSpecIntent.AnyOfArray = append(newAppSpecIntent.AnyOfArray, plcIntent)
					}
				}
				migParam.AppsNameDetails[gpIntentName][index].Phase = DeletePhase
				break
			case DeletePhase:
				// Skip primary placement intents
				migParam.AppsNameDetails[gpIntentName][index].Phase = ApplyPhase // TODO: is it necessary?
				break
			default:
				err := fmt.Errorf("error: %v is a bad phase", appNameDetails.Phase)
				fmt.Fprintf(os.Stderr, err.Error())
				return nil, err
			}

			appIntentURL := appIntentBaseURL + "/" + appNameDetails.AppIntentName
			newAppIntent := AppIntent{
				MetaData: MetaData{Name: appNameDetails.AppIntentName},
				Spec: SpecData{
					AppName: appNameDetails.AppName,
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

func CheckReadinessStatus(ctx context.Context, migParam MigParam) (*MigParam, error) {

	anchor, format, status, app, cluster := fillQueryParams(migParam)
	WatchGrpcEndpoint(migParam, anchor, format, status, app, cluster)

	return &migParam, nil
}

func buildDigURL(params map[string]string) string {
	url := params["emcoOrchEndpoint"]
	url += "/v2/projects/" + params["project"]
	url += "/composite-apps/" + params["compositeApp"]
	url += "/" + params["compositeAppVersion"]
	url += "/deployment-intent-groups/" + params["deploymentIntentGroup"]

	return url
}

func buildStatusAnchor(params map[string]string) string {
	url := "projects/" + params["project"]
	url += "/composite-apps/" + params["compositeApp"]
	url += "/" + params["compositeAppVersion"]
	url += "/deployment-intent-groups/" + params["deploymentIntentGroup"]
	url += "/status"

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

// checkIfSkipPrimaryIntentAllOf is used to make sure, that in the newly created placement intent
// there will not be any duplicated intents in AllOf array. If target Cluster was already present
// in the Primary Intent we can skip Primary Intent and still service continuity will be maintained
func checkIfSkipPrimaryIntentAllOf(mp MigParam, primaryIntent AllOf, newIntents []AllOf) (skip bool) {
	for _, newIntent := range newIntents {
		skip = false
		// If source Provider and target Provider are different, don't skip Primary Intent
		// If source Provider, target Provider and Cluster Name are the same, skip Primary Intent
		if newIntent.ProviderName != primaryIntent.ProviderName {
			continue
		} else if newIntent.ClusterName == primaryIntent.ClusterName && newIntent.ClusterName != "" {
			skip = true
			return
		}

		// If Primary Intent is based on Cluster Label, rather than on Cluster Name
		// Check if target Cluster isn't represented by that Cluster Label.
		// If it is, skip Primary Intent. Otherwise, DIG Update will fail.
		if primaryIntent.ClusterLabelName != "" {
			clmEndpoint := GetClmEndpoint(mp)
			provider := primaryIntent.ProviderName
			label := primaryIntent.ClusterLabelName

			url := fmt.Sprintf("%v/v2/cluster-providers/%v/clusters?label=%v", clmEndpoint, provider, label)

			respBody, _ := getHttpRespBody(url)

			var clusters []string
			if err := json.Unmarshal(respBody, &clusters); err != nil {
				decodeErr := fmt.Errorf("Failed to decode GET responde body for URL %s.\n"+
					"Decoder error: %#v\n", url, err)
				fmt.Fprintf(os.Stderr, decodeErr.Error())
			}

			for _, cluster := range clusters {
				if newIntent.ClusterName == cluster {
					skip = true
					fmt.Printf("Skipping: NewIntentName: %v already in clusters: %v covered by label: %v",
						newIntent.ClusterName, clusters, primaryIntent.ClusterLabelName)
					return
				}
			}
		}
	}

	return
}

// checkIfSkipPrimaryIntentAnyOf is used to make sure, that in the newly created placement intent
// there will not be any duplicated intents in AnyOf array. If target Cluster was already present
// in the Primary Intent we can skip Primary Intent and still service continuity will be maintained
func checkIfSkipPrimaryIntentAnyOf(mp MigParam, primaryIntent AnyOf, newIntents []AnyOf) (skip bool) {
	for _, newIntent := range newIntents {
		skip = false
		// If source Provider and target Provider are different, don't skip Primary Intent
		// If source Provider, target Provider and Cluster Name are the same, skip Primary Intent
		if newIntent.ProviderName != primaryIntent.ProviderName {
			continue
		} else if newIntent.ClusterName == primaryIntent.ClusterName && newIntent.ClusterName != "" {
			skip = true
			return
		}

		// If Primary Intent is based on Cluster Label, rather than on Cluster Name
		// Check if target Cluster isn't represented by that Cluster Label.
		// If it is, skip Primary Intent. Otherwise, DIG Update will fail.
		if primaryIntent.ClusterLabelName != "" {
			clmEndpoint := GetClmEndpoint(mp)
			provider := primaryIntent.ProviderName
			label := primaryIntent.ClusterLabelName

			url := fmt.Sprintf("http://%v/v2/cluster-providers/%v/clusters?label=%v", clmEndpoint, provider, label)

			respBody, _ := getHttpRespBody(url)

			var clusters []string
			if err := json.Unmarshal(respBody, &clusters); err != nil {
				decodeErr := fmt.Errorf("Failed to decode GET responde body for URL %s.\n"+
					"Decoder error: %#v\n", url, err)
				fmt.Fprintf(os.Stderr, decodeErr.Error())
			}

			for _, cluster := range clusters {
				if newIntent.ClusterName == cluster {
					skip = true
					fmt.Printf("Skipping: NewIntentName: %v already in clusters: %v covered by label: %v",
						newIntent.ClusterName, clusters, primaryIntent.ClusterLabelName)
					return
				}
			}
		}
	}

	return
}
