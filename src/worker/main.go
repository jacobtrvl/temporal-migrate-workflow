package main

// Worker process for the workflow.
// Registers app-specific workflow and activity code, then runs them.

import (
	"fmt"
	"gitlab.com/project-emco/samples/temporal/migrate-workflow/src/nvidiawf"
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"gitlab.com/project-emco/samples/temporal/migrate-workflow/src/emcomigrate"
)

const (
	temporal_env_var = "TEMPORAL_SERVER"
	temporal_port    = "7233"
)

func main() {
	// Get the Temporal Server's IP
	temporal_server := os.Getenv(temporal_env_var)
	if temporal_server == "" {
		fmt.Fprintf(os.Stderr, "Error: Need to define $TEMPORAL_SERVER\n")
		os.Exit(1)
	}
	hostPort := temporal_server + ":" + temporal_port
	fmt.Printf("Temporal server endpoint: (%s)\n", hostPort)

	// Create the client object just once per process
	options := client.Options{HostPort: hostPort}
	c, err := client.NewClient(options)
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer c.Close()

	// Create worker for DU migration
	w1 := worker.New(c, nvidiawf.NwfTaskQueue, worker.Options{})
	w1.RegisterWorkflow(nvidiawf.NvidiaWorkflow)
	w1.RegisterActivity(nvidiawf.DoDigApprove)
	w1.RegisterActivity(nvidiawf.DoDigInstantiate)
	w1.RegisterActivity(nvidiawf.DoDigTerminate)

	err = w1.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start nvidia worder", err)
	}
	fmt.Printf("Nv WF started : (%s)\n", hostPort)

	// This worker hosts both Workflow and Activity functions
	w := worker.New(c, emcomigrate.MigTaskQueue, worker.Options{})
	w.RegisterWorkflow(emcomigrate.EmcoMigrateWorkflow)
	w.RegisterActivity(emcomigrate.GetDigAppIntents)
	w.RegisterActivity(emcomigrate.UpdateAppIntents)
	w.RegisterActivity(emcomigrate.DoDigUpdate)

	// Start listening to the Task Queue
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start Worker", err)
	}

}
