```
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2022 Intel Corporation, Orange S.A.
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

# Edge Relocation

`Relocate-Workflow` is designed as Temporal Workflow, which make use of EMCO APIs to perform E2E application relocation 
in a seamless way. Ultimately, `Relocate-Workflow` should allow `zero-down-time` relocation.

## Problem statement

The user (UE) is consuming a service, while moving out of the coverage area of Source MEC Host (Edge Server / `Cluster A`). Later user enters the coverage area of Target MEC (Edge Server / `Cluster B`) and expects to resume the same service. This requires a relocation of a service instance from `Cluster A` to `Cluster B`

## Major requirements

- Service continuity must be assured to the UE;
- The new instance of the application must be declared to be 'ready’ before we can steer the trafic to the new app instance;
- If there are several candidates for the target MEC cluster, the final choice should be made by MEC Orchestrator.

## 3 step to make end-to-end Edge Relocation happend

1. UE monitoring
2. Decision about relocation
3. **Perform Edge Relocation with Zero Downtime (`Relocate-Workflow`)**