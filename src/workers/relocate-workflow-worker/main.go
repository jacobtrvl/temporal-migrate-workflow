package main

// Worker process for the workflow.
// Registers app-specific workflow and activity code, then runs them.

import (
	"fmt"
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"gitlab.com/project-emco/samples/temporal/migrate-workflow/src/emcorelocate"
)

const (
	TemporalIpEnvVar   = "TEMPORAL_SERVER_IP"
	temporalPortEnvVar = "TEMPORAL_SERVER_PORT" // 7233
)

func main() {
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

	// Create the client object just once per process
	options := client.Options{HostPort: hostPort}
	c, err := client.NewClient(options)
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer c.Close()
	// This worker hosts both Workflow and Activity functions
	w := worker.New(c, emcorelocate.MigTaskQueue, worker.Options{})
	w.RegisterWorkflow(emcorelocate.EmcoRelocateWorkflow)
	w.RegisterActivity(emcorelocate.GetDigAppIntents)
	w.RegisterActivity(emcorelocate.CheckReadinessStatus)
	w.RegisterActivity(emcorelocate.UpdateAppIntents)
	w.RegisterActivity(emcorelocate.DoDigUpdate)

	// Start listening to the Task Queue
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start Worker", err)
	}
}
