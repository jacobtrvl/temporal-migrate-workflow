// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package nvidiawf

import (
	"fmt"
	"os"
	"time"

	eta "gitlab.com/project-emco/core/emco-base/src/workflowmgr/pkg/emcotemporalapi"
	wf "go.temporal.io/sdk/workflow"
)

// special name that matches all activities
const ALL_ACTIVITIES = "all-activities"

// Treat this as a const
var NeededParams = []string{ // parameters needed for this workflow
	"emcoURL", "project", "compositeApp", "compositeAppVersion", "deploymentIntentGroup",
	"compositeApp1", "compositeAppVersion1", "deploymentIntentGroup1"}

// NvidiaWorklow is a Temporal workflow that terminates a DIG in one cluster
// and instantiates a DIG in another cluster followed by a fronthaul switch configuration
func NvidiaWorkflow(ctx wf.Context, wfParam *eta.WorkflowParams) (*NwfParam, error) {
	// List all activities for this workflow
	activityNames := []string{
		"DoDigInstantiate",
		"DoDigTerminate",
	}

	// Set current state and define workflow queries
	currentState := "started" // name of ongoing activity, "started" or "completed"
	queryType := "current-state"
	err := wf.SetQueryHandler(ctx, queryType, func() (string, error) {
		return currentState, nil
	})
	if err != nil {
		currentState = "failed to register current state query handler"
		return nil, err
	}

	// Check that we got "all-activities" params
	all_activities_params, ok := wfParam.ActivityParams[ALL_ACTIVITIES]
	if !ok {
		err := fmt.Errorf("NvidiaWorkflow: expect %s parameters", ALL_ACTIVITIES)
		fmt.Fprintf(os.Stderr, err.Error())
		return nil, err
	}

	if err := validateParams(all_activities_params); err != nil {
		return nil, err
	}

	// Print activity options from the workflow parameters.
	optsMap := wfParam.ActivityOpts
	actsWithOpts := make([]string, 0, len(optsMap))
	for actName := range optsMap {
		actsWithOpts = append(actsWithOpts, actName)
	}
	fmt.Printf("NvidiaWorkflow: got activity options for %#v\n", actsWithOpts)

	// Create a separate context for each activity based on its activity options.
	ctxMap, err := getActivityContextMap(ctx, activityNames, optsMap)
	if err != nil {
		return nil, err
	}

	params := NwfParam{InParams: all_activities_params}

	currentState = "DoDigInstantiate"
	ctx1 := ctxMap["DoDigInstantiate"]
	err = wf.ExecuteActivity(ctx1, DoDigInstantiate, params).Get(ctx1, &params)
	if err != nil {
		wferr := fmt.Errorf("DoDigInstantiate failed: %s", err.Error())
		fmt.Fprintf(os.Stderr, wferr.Error())
		return nil, wferr
	}

	currentState = "DoDigTerminate"
	ctx2 := ctxMap["DoDigTerminate"]
	err = wf.ExecuteActivity(ctx2, DoDigTerminate, params).Get(ctx2, &params)
	if err != nil {
		wferr := fmt.Errorf("DoDigTerminate failed: %s", err.Error())
		fmt.Fprintf(os.Stderr, wferr.Error())
		return nil, wferr
	}
	currentState = "completed"

	fmt.Printf("After all activities: migParam = %#v\n", params)

	return &params, nil
}

// getActivityContextMap returns a list of Temporal contexts for each activity.
// Note that this is generic code that is independent of user's app/workflows.
func getActivityContextMap(ctx wf.Context, activityNames []string,
	optsMap map[string]wf.ActivityOptions) (map[string]wf.Context, error) {

	// Validate that all activity names in given workflow params are valid
	all_activities_flag := false
	for paramActName := range optsMap {
		found := false
		for _, actName := range activityNames {
			if paramActName == actName {
				found = true
				break
			}
			if paramActName == ALL_ACTIVITIES {
				found = true
				all_activities_flag = true
				break
			}
		}
		if !found {
			err := fmt.Errorf("Invalid activity name in params: %s", paramActName)
			return nil, err
		}
	}

	// Init each activity-specific context to default or the specified param
	defaultActivityOpts := wf.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}
	ctxMap := make(map[string]wf.Context, len(activityNames))
	for _, actName := range activityNames {
		// init to default
		ctxMap[actName] = wf.WithActivityOptions(ctx, defaultActivityOpts)
		// Apply all-activities options if specified
		if all_activities_flag {
			fmt.Printf("Applying all-activities options for activity %s\n", actName)
			ctxMap[actName] = wf.WithActivityOptions(ctx, optsMap[ALL_ACTIVITIES])
		}
		// Apply activity-specific options, if specified
		for paramActName := range optsMap {
			if paramActName == actName {
				fmt.Printf("Applying activity-specifc options for %s\n", actName)
				ctxMap[actName] = wf.WithActivityOptions(ctx, optsMap[actName])
			}
		}
	}
	return ctxMap, nil
}

// validateParams verifies that inParams has all needed params for this workflow
func validateParams(inParams map[string]string) error {

	paramsNotFound := []string{}
	for _, neededParam := range NeededParams {
		found := false
		for inParam := range inParams {
			if neededParam == inParam {
				found = true
			}
		}
		if !found {
			paramsNotFound = append(paramsNotFound, neededParam)
		}
	}

	if len(paramsNotFound) > 0 {
		err := fmt.Errorf("Workflow needs these params: %#v\n", paramsNotFound)
		fmt.Fprintf(os.Stderr, err.Error())
		return err
	}

	return nil
}
