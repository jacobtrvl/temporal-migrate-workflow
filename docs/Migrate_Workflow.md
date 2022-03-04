```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2022 Intel Corporation
```
<!-- omit in toc -->
# Migrate: A Temporal Workflow in EMCO for App Migration

This document describes a reference Temporal workflow designed to be used
with EMCO. The workflow migrates a stateless application deployed by EMCO
in one cluster to another specified cluster. 

The reader is expected to be familiar with
[EMCO](https://gitlab.com/project-emco/core/emco-base) and
[Temporal](https://docs.temporal.io/docs/temporal-explained/introduction).
In particular, it is important to read the document [Temporal Workflows in
EMCO](https://gitlab.com/project-emco/core/emco-base/-/blob/emco-temporal/docs/user/Temporal_Workflows_In_EMCO.md) first.

## Introduction
The Edge Multi-Cluster Orchestrator (EMCO), an open source project in Linux
Foundation Networking, has been enhanced to launch and manage Temporal
workflows. This repository contains a reference workflow that migrates
a stateless application deployed by EMCO in one cluster to another
specified cluster. This can be taken as a template to develop workflows to
migrate stateful applications and other workflows as well.

## Reference Workflow For EMCO
The relationship among the main workflow entities is explained in the document
[Temporal Workflows in EMCO](https://gitlab.com/project-emco/core/emco-base/-/blob/emco-temporal/docs/user/Temporal_Workflows_In_EMCO.md). It is recapitulated below.

In general, a workflow is executed by a worker entity within a worker process;
there can be one or more worker entities within a worker process, and one or
more worker processes in a system. In this specific migration workflow, there
is one worker process with one worker entity, which executes one workflow
with three activities.

In general, there can be multiple workflow clients managed by EMCO. Each
client may launch many workflows, including copies of the same workflow,
with distinct workflow IDs. In the migration workflow, there is only one
workflow client and it starts only one workflow. But the source code
layout, build environment and the workflow container image can all be
extended to multiple workflow clients.

The atructure of the workflow client container can take any form: EMCO does
not mandate anything. In the migration workflow, the workflow client
container has a HTTP server that receives a `HTTP POST` call from EMCO's
`workflowmgr` microservice and executes the named workflow client binary
with the named workflow ID. This design allows for many workflow clients
to be packaged with a single HTTP server, and the `POST` call specifies
which workflow client needs to be executed.

The communication between the workers and the workflow clients can take any
form: EMCO has no specific requirements. In the migration workflow, both
the worker container and the workflow container get an environment variable
specifying the network endpoint of the Temporal Server.

## Source Code Layout
The repository includes both a workflow client and the workflow per se. 
Each of those can be conceptually divided into generic Temporal code
and workflow-specific code. The source code layout reflects those
categories. Under `src`, we have:

 * `workflowclients/`: code related to workflow client(s).
   * `http_server/`: The common HTTP server for all workflow clients.
   * `migrate_workflowclient/`: The only workflow client available now.
   * Can add more directories for other workflow clients in the future.
 * `worker/`: The worker process for migrate workflow.
 * `emcomigrate/`:  The core workflow and activities for migration.

## Demo Steps

* Deploy EMCO.

* Deploy the sample application n `samples/apps` using EMCO: see `samples/intents`.

* Build the workflow client and workflow container images.

* Deploy the Temporal server.

* Deploy the workflow client and workflow container images.

* Define the workflow intent in EMCO.

* Run the workflow start API.

* Optionally run te status query and cancel APIs.

## Resources
 * [Temporal workfow engine](https://docs.temporal.io/docs/temporal-explained/introduction)

