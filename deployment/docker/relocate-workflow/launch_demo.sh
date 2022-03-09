#!/bin/bash

run_worker() {
   local worker_img="relocate-workflow-worker:latest"
   local net="--network=emconet"
   local envargs="--env TEMPORAL_SERVER=${TEMPORAL_SERVER}"
   local args="-d" # "-it"

   docker run --rm ${net} ${envargs} ${args} ${worker_img}
}

run_workflowclient() {
   local wfclient_img="workflow-client:latest"
   local net="--network=emconet"
   local envargs="--env TEMPORAL_SERVER=${TEMPORAL_SERVER}"
   local args="-d" # "-it"
   local port="--expose 9090"

   docker run --rm ${net} ${envargs} ${args} ${port} ${wfclient_img}
}

run_worker
sleep 2
run_workflowclient
