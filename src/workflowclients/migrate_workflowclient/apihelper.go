// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	eta "gitlab.com/project-emco/core/emco-base/src/workflowmgr/pkg/emcotemporalapi"
)

func getTemporalSpec(filename string) (*eta.WfTemporalSpec, error) {
	var spec eta.WfTemporalSpec
	var err error

	argData := []byte("{}")
	if filename != "" {
		argData, err = ioutil.ReadFile(filename)
		if err != nil {
			// stdout and stderr are both reported by http server process.
			fmt.Fprintln(os.Stderr, err)
			return &eta.WfTemporalSpec{}, err
		}
		fmt.Printf("%s\n", string(argData))
	}

	if err := json.Unmarshal(argData, &spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return &eta.WfTemporalSpec{}, err
	}
	fmt.Printf("raw spec: %#v\n", spec)

	// TODO Validate spec?

	return &spec, nil
}

func validateSpec(spec *eta.WfTemporalSpec) error {
	if spec.WfStartOpts.ID == "" {
		err := fmt.Errorf("Error: Need to provide a name in " +
			"spec.workflowStartOptions.id")
		return err
	}
	if spec.WfStartOpts.TaskQueue != "" {
		warn := fmt.Errorf("Warning: Ignoring task queue name in " +
			"spec.workflowStartOptions.taskQueue")
		fmt.Fprintln(os.Stderr, warn.Error())
		// Warning, not an error
	}
	return nil
}
