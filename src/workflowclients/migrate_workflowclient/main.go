package main

import (
	"context"
	"fmt"
	"log"

	//"encoding/json"
	"flag"
	"os"

	"go.temporal.io/sdk/client"

	eta "gitlab.com/project-emco/core/emco-base/src/workflowmgr/pkg/emcotemporalapi"
	"gitlab.com/project-emco/samples/temporal/migrate-workflow/src/emcomigrate"
)

const (
	TemporalIpEnvVar   = "TEMPORAL_SERVER_IP"
	temporalPortEnvVar = "TEMPORAL_SERVER_PORT" // 7233
)

func main() {
	var argFileName string
	var spec *eta.WfTemporalSpec

	// Get the Temporal Server's IP
	temporalServerIp := os.Getenv(TemporalIpEnvVar)
	if temporalServerIp == "" {
		fmt.Fprintf(os.Stderr, "Error: Need to define $TEMPORAL_SERVER_IP\n")
		os.Exit(1)
	}

	// Get the TemporalServer's Port
	temporalServerPort := os.Getenv(temporalPortEnvVar)
	if temporalServerPort == "" {
		fmt.Fprintf(os.Stderr, "Error: Need to define $TEMPORAL_SERVER_PORT\n")
		os.Exit(1)
	}

	hostPort := temporalServerIp + ":" + temporalServerPort
	fmt.Printf("Temporal server endpoint: (%s)\n", hostPort)

	// Get the JSON arg
	flag.StringVar(&argFileName, "a", "", "Workflow params as JSON file")
	flag.Parse()
	if argFileName != "" {
		fmt.Printf("Will read parameters from file: %s\n", argFileName)
	}

	spec, err := getTemporalSpec(argFileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Quitting due to errors.\n")
	}
	spec.WfStartOpts.TaskQueue = emcomigrate.MigTaskQueue //override task queue

	// Create the client object just once per process
	clientOptions := client.Options{HostPort: hostPort}
	c, err := client.NewClient(clientOptions)
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer c.Close()

	// NOTE: This cast assumes Temporal's StartWorkflowOptions == EMCO's version.
	options := client.StartWorkflowOptions(spec.WfStartOpts)
	//migParam := emcomigrate.MigParam{Name: "Ganesha"}
	we, err := c.ExecuteWorkflow(context.Background(), options,
		emcomigrate.EmcoMigrateWorkflow, &spec.WfParams)
	if err != nil {
		log.Fatalln("error starting Migration Workflow", err)
	}
	log.Printf("\nFinished workflow. WorkflowID: %s RunID: %s\n", we.GetID(), we.GetRunID())
}
