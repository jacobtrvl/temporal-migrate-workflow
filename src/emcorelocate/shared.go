// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package emcorelocate

const MigTaskQueue = "RELOCATE_TASK_Q"

type AppNameDetails struct {
	AppName       string
	AppIntentName string
	//TODO: replace phase var to handle enum instead of plain string
	Phase         string
	PrimaryIntent IntentStruc
}

type MigParam struct {
	InParams                  map[string]string
	GenericPlacementIntentURL string
	GenericPlacementIntents   []string
	// map indexed by generic placement intent name
	AppsNameDetails map[string][]AppNameDetails
}
