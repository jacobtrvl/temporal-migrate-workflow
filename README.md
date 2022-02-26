```
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2022 Intel Corporation
```

# Sample Temporal Workflow For EMCO

This repo is meant to illustrate how a Temporal workflow can be used
with EMCO. It contains these components:
 * Code for a sample workflow that migrates a stateless application
   deployed by EMCO in one Kubernetes cluster to another Kubernetes
   cluster.
 * Code for the Temporal worker that executes the workflow and for the
   workflow client that invokes the workflow.
 * A build environment to build the binaries and Docker container images
   for the worker and the workflow client.
 * A deployment environment to run the worker and the workflow client,
   either locally via docker-compose or in a remote Kubernetes cluster via
   Helm (and optionally EMCO).

# Prerequisites

NOTE: EMCO-Temporal integration in `emco-base` is WIP. Till that completes,
  you will first need to clone the `emco-temporal` branch of `emco-base`
  into a local directory that is a sibling directory of this repository.
  That is done as below:
  ```
  $ cd .. # change PWD to parent directory of temporal-migrate-workflow repo.
  $ git clone https://gitlab.com/project-emco/core/emco-base.git -b emco-temporal
  ```
