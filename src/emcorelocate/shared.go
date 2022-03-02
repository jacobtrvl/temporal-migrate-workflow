// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package emcorelocate

const MigTaskQueue = "MIGRATION_TASK_Q"

type AppNameIntentPair struct {
	AppName       string
	AppIntentName string
}

type MigParam struct {
	InParams                  map[string]string
	GenericPlacementIntentURL string
	GenericPlacementIntents   []string
	// map indexed by generic placement intent name
	AppNameIntentPairs map[string][]AppNameIntentPair
}
