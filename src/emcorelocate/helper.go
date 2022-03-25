// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package emcorelocate

import (
	"fmt"
	"google.golang.org/grpc"
	"strings"
	"time"

	"context"
	"github.com/google/uuid"
	statusnotifypb "gitlab.com/project-emco/samples/temporal/migrate-workflow/src/statusnotify"
)

const MigTaskQueue = "RELOCATE_TASK_Q"

type UpdatePhase int8

const (
	ApplyPhase UpdatePhase = iota
	DeletePhase
)

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
// TODO: This is done only for testing purposes. Please remove hardcoded Status Endpoint
func GetOrchestratorGrpcEndpoint(mp MigParam) string {
	return mp.InParams["emcoOrchStatusEndpoint"]
}

// GetClmEndpoint is endpoint for cluster manager microservice
// TODO: This is done only for testing purposes. Please remove hardcoded Status Endpoint
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

	//fill querry params
	reg.Output = statusnotifypb.OutputType_ALL
	reg.StatusType = statusnotifypb.StatusValue_READY
	reg.Apps = make([]string, 0)
	reg.Clusters = make([]string, 0)
	reg.Resources = make([]string, 0)

	reg.Apps = append(reg.Apps, mp.InParams["targetAppName"])
	reg.Clusters = append(reg.Clusters, fmt.Sprintf("%s+%s", mp.InParams["targetClusterProvider"], mp.InParams["targetClusterName"]))

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
