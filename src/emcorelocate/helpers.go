// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package emcorelocate

import (
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"context"
	"github.com/google/uuid"
	statusnotifypb "gitlab.com/project-emco/samples/temporal/migrate-workflow/src/statusnotify"
)

const MigTaskQueue = "RELOCATE_TASK_Q"

const (
	ApplyPhase UpdatePhase = iota
	DeletePhase
)

type UpdatePhase int8

type AppNameDetails struct {
	AppName       string
	AppIntentName string
	Phase         UpdatePhase
	PrimaryIntent IntentStruc
}

type MigParam struct {
	InParams                  map[string]string
	StatusAnchor              string
	GenericPlacementIntentURL string
	GenericPlacementIntents   []string
	// map indexed by generic placement intent name
	AppsNameDetails map[string][]AppNameDetails
}

// GetOrchestratorGrpcEndpoint gRPC endpoint for Orchestrator
func GetOrchestratorGrpcEndpoint(mp MigParam) string {
	return mp.InParams["emcoOrchStatusEndpoint"]
}

// GetClmEndpoint is endpoint for cluster manager microservice
func GetClmEndpoint(mp MigParam) string {
	return mp.InParams["emcoClmEndpoint"]
}

// WatchGrpcEndpoint reads the configuration file to get gRPC Endpoint
// and makes a connection to watch status notifications.
func WatchGrpcEndpoint(mp MigParam) {
	var endpoint string
	var anchor string
	var reg statusnotifypb.StatusRegistration

	anchor = buildStatusAnchor(mp.InParams)
	fmt.Printf("\nCheckReadinessStatus: statusAnchor = %s\n", anchor)

	// fill query params
	reg.Output = statusnotifypb.OutputType_SUMMARY
	reg.StatusType = statusnotifypb.StatusValue_READY
	reg.Apps = make([]string, 0)
	reg.Clusters = make([]string, 0)
	reg.Resources = make([]string, 0)

	reg.Apps = append(reg.Apps, mp.InParams["targetAppName"])
	reg.Clusters = append(reg.Clusters,
		fmt.Sprintf("%s+%s", mp.InParams["targetClusterProvider"], mp.InParams["targetClusterName"]))

	reg.Resources = append(reg.Resources, "apps.v1.Deployment", "apps.v1.StatefulSet", "apps.v1.DaemonSet",
		"apps.v1.ReplicaSet", "batch.v1.Job", "v1.Pod")

	s := strings.Split(anchor, "/")
	if len(s) < 1 {
		fmt.Errorf("Invalid Anchor: %s\n", s)
		return
	}

	switch s[0] {
	case "projects":
		if len(s) == 8 && s[2] == "composite-apps" && s[5] == "deployment-intent-groups" && s[7] == "status" {
			endpoint = GetOrchestratorGrpcEndpoint(mp)
			reg.Key = &statusnotifypb.StatusRegistration_DigKey{
				DigKey: &statusnotifypb.DigKey{
					Project:               s[1],
					CompositeApp:          s[3],
					CompositeAppVersion:   s[4],
					DeploymentIntentGroup: s[6],
				},
			}
			break
		}
		fmt.Errorf("Invalid status anchor: %s\n", s)
		return
	default:
		fmt.Errorf("Invalid status anchor: %s\n", s)
		return
	}

	reg.ClientId = uuid.New().String()

	conn, err := newGrpcClient(endpoint)
	if err != nil {
		fmt.Errorf("Error connecting to gRPC status endpoint: %s, Error: %s\n", endpoint, err)
		return
	}

	client := statusnotifypb.NewStatusNotifyClient(conn)

	stream, err := client.StatusRegister(context.Background(), &reg, grpc.WaitForReady(true))
	if err != nil {
		fmt.Errorf("Error registering for status notifications, gRPC status endpoint: %s, Error: %s\n", endpoint, err)
		return
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			fmt.Errorf("error reading from status stream: %s\n", err)
			time.Sleep(5 * time.Second) // protect against potential deluge of errors in the for loop
			continue
		}
		fmt.Printf("CheckReadinessStatus: StatusValue is %v\n", resp.StatusValue.String())
		if resp.StatusValue.String() == "READY" {
			if err := conn.Close(); err != nil {
				fmt.Errorf("error wile closing conn: %s\n", err)
			}
			break
		}
	}
}

func buildStatusAnchor(params map[string]string) string {
	url := "projects/" + params["project"]
	url += "/composite-apps/" + params["compositeApp"]
	url += "/" + params["compositeAppVersion"]
	url += "/deployment-intent-groups/" + params["deploymentIntentGroup"]
	url += "/status"

	return url
}

// CreateGrpcClient creates the gRPC Client Connection
func newGrpcClient(endpoint string) (*grpc.ClientConn, error) {
	var err error
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		fmt.Printf("Grpc Client Initialization failed with error: %v\n", err)
	}

	return conn, err
}

func buildDigURL(params map[string]string) string {
	url := params["emcoOrchEndpoint"]
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

// TODO: Use Generics in the future
// checkIfSkipPrimaryIntentAllOf is used to make sure, that in the newly created placement intent
// there will not be any duplicated intents in AllOf array. If target Cluster was already present
// in the Primary Intent we can skip Primary Intent and still service continuity will be maintained
func checkIfSkipPrimaryIntentAllOf(mp MigParam, primaryIntent AllOf, newIntents []AllOf) (skip bool) {
	for _, newIntent := range newIntents {
		skip = false
		if newIntent.ProviderName != primaryIntent.ProviderName {
			continue
		} else if newIntent.ClusterName == primaryIntent.ClusterName && newIntent.ClusterName != "" {
			skip = true
			return
		}
		// If Primary Intent is based on Cluster Label, rather than on Cluster Name. Check if target Cluster isn't
		// represented by that Cluster Label. If it is, skip Primary Intent. Otherwise, DIG Update will fail.
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

// TODO: Use Generics in the future
// checkIfSkipPrimaryIntentAnyOf is used to make sure, that in the newly created placement intent
// there will not be any duplicated intents in AnyOf array. If target Cluster was already present
// in the Primary Intent we can skip Primary Intent and still service continuity will be maintained
func checkIfSkipPrimaryIntentAnyOf(mp MigParam, primaryIntent AnyOf, newIntents []AnyOf) (skip bool) {
	for _, newIntent := range newIntents {
		skip = false
		if newIntent.ProviderName != primaryIntent.ProviderName {
			continue
		} else if newIntent.ClusterName == primaryIntent.ClusterName && newIntent.ClusterName != "" {
			skip = true
			return
		}
		// If Primary Intent is based on Cluster Label, rather than on Cluster Name. Check if target Cluster isn't
		// represented by that Cluster Label. If it is, skip Primary Intent. Otherwise, DIG Update will fail.
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
