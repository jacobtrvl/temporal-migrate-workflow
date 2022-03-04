# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2022 Intel Corporation
export GO111MODULE=on
export EMCOBUILDROOT=$(shell pwd)
export CONFIG := $(wildcard config/*.txt)

all: build-docker-worker build-docker-workflowclient

build-docker-worker: compile-worker
	@echo "Building worker container"
	docker build --rm -f build/docker/Dockerfile.relocateWorker -t relocate-workflow-worker ./bin/relocate-worker

compile-worker:
	@echo "Compiling worker with app"
	@mkdir -p bin/relocate-worker
	@cd src/relocate-worker; go build -o ../../bin/relocate-worker/relocate-worker main.go

clean-worker:
	/bin/rm -rf bin/relocate-worker

build-docker-workflowclient: compile-workflowclient
	@echo "Building workflowclient container"
	docker build --rm -f build/docker/Dockerfile.workflowclient -t workflow-client ./bin/workflowclients

compile-workflowclient:
	@echo "Compiling workflowclients"
	@mkdir -p bin/workflowclients
	@cd src/workflowclients; \
		go build -o ../../bin/workflowclients/migrate_workflowclient migrate_workflowclient/*.go && \
		go build -o ../../bin/workflowclients/relocate_workflowclient relocate_workflowclient/*.go && \
		go build -o ../../bin/workflowclients/http_server http_server/main.go;

clean-workflowclient:
	/bin/rm -rf bin/workflowclients

clean: clean-worker clean-workflowclient

test:
	@echo "No tests yet"

tidy:
	@echo "No dependencies to clean"

