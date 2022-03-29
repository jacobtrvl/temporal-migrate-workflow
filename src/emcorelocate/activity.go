// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package emcorelocate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

		// TODO []AppNameDetails doesn't have to be a list. We want to relocate just 1 application.
		// leave [] for now to maintain compatibility with migrate workflow
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
		// TODO: We could use "equal", rather than "less than". We relocate just 1 app.
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

	// TODO: Consider clusterLabel
	// newAppSpecIntent is target intent for relocated app. For now we assume, that only cluster name can be used.
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

	WatchGrpcEndpoint(migParam)
	return &migParam, nil
}
