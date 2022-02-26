// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const (
	name       = "workflow-listener"
	execDir    = "/opt/emco"
	invokerURL = "/invoke/{wfclient:[a-zA-Z0-9-_]+}" // URL to invoke the workflow client
	httpPort   = "9090"
)

// runWorkflowClient runs the workflow client named by the URL.
//  The URL is expected to be of the form /invoke/$workflow_client_name .
//  The executable binary for the workflow client must be in execDir.
func runWorkflowClient(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		wrapErr := fmt.Errorf("POST body read err; %v\n", err)
		log.Printf(wrapErr.Error())
		http.Error(w, wrapErr.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("POST body: %s", string(body))

	params := mux.Vars(r)
	wfClientName := params["wfclient"]

	// Create a temp file, in /tmp by default.
	// NOTE: Go replaces "*" in the name with a random number.
	tmpfile, err := ioutil.TempFile("", wfClientName+".*.json")
	if err != nil {
		wrapErr := fmt.Errorf("Failed to create temp file for %s", wfClientName)
		log.Printf(wrapErr.Error())
		http.Error(w, wrapErr.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Created file: %s", tmpfile.Name())

	// Write POST body to the temp file.
	if err = ioutil.WriteFile(tmpfile.Name(), body, 0444); err != nil {
		wrapErr := fmt.Errorf("Failed to write POST body to temp file %s\n"+
			"Error: %s\n", tmpfile.Name(), err)
		log.Printf(wrapErr.Error())
		http.Error(w, wrapErr.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Wrote POST body to file: %s", tmpfile.Name())

	// Eexcute the workflow client command.
	wfClient := path.Join(execDir, wfClientName)
	log.Printf("Will execute: (%s -a %s)\n", wfClient, tmpfile.Name())

	cmd := exec.Command(wfClient, "-a", tmpfile.Name())
	cmdOutErr, err := cmd.CombinedOutput()
	if err != nil {
		wrapErr := fmt.Errorf("%s finished with error: %v\n", wfClient, err)
		log.Printf(wrapErr.Error())
		http.Error(w, wrapErr.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("\nOutput from %s :\n%s\n", wfClient, cmdOutErr)
	w.WriteHeader(http.StatusNoContent)
}

// NewRouter creates a router that registers the various urls that are supported
func NewRouter() *mux.Router {

	router := mux.NewRouter()

	router.HandleFunc(invokerURL, runWorkflowClient).Methods("POST")

	return router
}

func main() {
	httpRouter := NewRouter()
	loggedRouter := handlers.LoggingHandler(os.Stdout, httpRouter)
	log.Println("Starting http server")

	httpServer := &http.Server{
		Handler: loggedRouter,
		Addr:    ":" + httpPort,
	}

	connectionsClose := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		httpServer.Shutdown(context.Background())
		close(connectionsClose)
	}()

	err := httpServer.ListenAndServe()
	log.Printf("httpServer returned: %s\n", err)
}
