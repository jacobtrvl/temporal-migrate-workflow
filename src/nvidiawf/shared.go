// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Intel Corporation

package nvidiawf

import "sync"

const NwfTaskQueue = "NVIDIAWF_TASK_Q"

type NwfParam struct {
	InParams map[string]string
	App      string
	mu       sync.Mutex
}
