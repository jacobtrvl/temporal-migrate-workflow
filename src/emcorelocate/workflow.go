package emcorelocate

import (
	"fmt"
	"os"
	"time"

	eta "gitlab.com/project-emco/core/emco-base/src/workflowmgr/pkg/emcotemporalapi"
	wf "go.temporal.io/sdk/workflow"
)

// special name that matches all activities
const ALL_ACTIVITIES = "all-activities"

// NeededParams should be treated as a const
var NeededParams = []string{ // parameters needed for this workflow
	"emcoOrchEndpoint", "project", "compositeApp", "compositeAppVersion", "deploymentIntentGroup",
	"targetClusterProvider", "targetClusterName", "targetAppName"}

// EmcoRelocateWorkflow is a Temporal workflow that relocates selected app of a
// given deployment intent group (DIG) to a given target cluster in zero down-time mode.
// It means that new app instance will be in 'ready' STATE before old app instance will be deleted.
// It expects an "all-activities" parameter inside wfParam.InParams that
// specifies the common retry/timeout policies for all activities. It may
// have other activity-specific options on top of that.
func EmcoRelocateWorkflow(ctx wf.Context, wfParam *eta.WorkflowParams) (*MigParam, error) {
	// List all activities for this workflow
	activityNames := []string{
		"GetDigAppIntents",
		"UpdateAppIntents",
		"DoDigUpdate",
		"CheckReadinessStatus",
		"UpdateAppIntents",
		"DoDigUpdate",
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
		err := fmt.Errorf("EmcoRelocateWorkflow: expect %s parameters", ALL_ACTIVITIES)
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
	fmt.Printf("EmcoRelocateWorkflow: got activity options for %#v\n", actsWithOpts)

	// Create a separate context for each activity based on its activity options.
	ctxMap, err := getActivityContextMap(ctx, activityNames, optsMap)
	if err != nil {
		return nil, err
	}

	migParam := MigParam{InParams: all_activities_params}

	currentState = "GetDigAppIntents"
	ctx1 := ctxMap["GetDigAppIntents"]
	err = wf.ExecuteActivity(ctx1, GetDigAppIntents, migParam).Get(ctx1, &migParam)
	if err != nil {
		wferr := fmt.Errorf("GetDigAppIntents failed: %s", err.Error())
		fmt.Fprintf(os.Stderr, wferr.Error())
		return nil, wferr
	}

	currentState = "UpdateAppIntents"
	ctx2 := ctxMap["UpdateAppIntents"]
	err = wf.ExecuteActivity(ctx2, UpdateAppIntents, migParam).Get(ctx2, &migParam)
	if err != nil {
		wferr := fmt.Errorf("UpdateAppIntents failed: %s", err.Error())
		fmt.Fprintf(os.Stderr, wferr.Error())
		return nil, wferr
	}

	currentState = "DoDigUpdate"
	ctx3 := ctxMap["DoDigUpdate"]
	err = wf.ExecuteActivity(ctx3, DoDigUpdate, migParam).Get(ctx3, &migParam)
	if err != nil {
		wferr := fmt.Errorf("DoDigUpdate failed: %s", err.Error())
		fmt.Fprintf(os.Stderr, wferr.Error())
		return nil, wferr
	}

	currentState = "CheckReadinessStatus"
	ctx4 := ctxMap["CheckReadinessStatus"]
	err = wf.ExecuteActivity(ctx4, CheckReadinessStatus, migParam).Get(ctx4, &migParam)
	if err != nil {
		wferr := fmt.Errorf("CheckReadinessStatus failed: %s", err.Error())
		fmt.Fprintf(os.Stderr, wferr.Error())
		return nil, wferr
	}

	currentState = "UpdateAppIntents"
	ctx5 := ctxMap["UpdateAppIntents"]
	err = wf.ExecuteActivity(ctx5, UpdateAppIntents, migParam).Get(ctx5, &migParam)
	if err != nil {
		wferr := fmt.Errorf("UpdateAppIntents failed: %s", err.Error())
		fmt.Fprintf(os.Stderr, wferr.Error())
		return nil, wferr
	}

	currentState = "DoDigUpdate"
	ctx6 := ctxMap["DoDigUpdate"]
	err = wf.ExecuteActivity(ctx6, DoDigUpdate, migParam).Get(ctx6, &migParam)
	if err != nil {
		wferr := fmt.Errorf("DoDigUpdate failed: %s", err.Error())
		fmt.Fprintf(os.Stderr, wferr.Error())
		return nil, wferr
	}

	currentState = "completed"

	fmt.Printf("After all activities: migParam = %#v\n", migParam)

	return &migParam, nil
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
