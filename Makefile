# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2022 Intel Corporation
export GO111MODULE=on
export EMCOBUILDROOT=$(shell pwd)
export CONFIG := $(wildcard config/*.txt)

ifndef WORKFLOWS
WORKFLOWS=migrate relocate
endif

all: build-docker-workers build-docker-workflowclient

build-docker-workers: compile-workers
	@echo "Building worker containers"
	@for w in $(WORKFLOWS); do \
		docker build --rm -f build/docker/Dockerfile.$$w-workflow-worker -t "$$w-workflow-worker" ./bin/$$w-workflow-worker; \
	done

compile-workers:
	@echo "Compiling workers with app"
	@for w in $(WORKFLOWS); do \
		/bin/mkdir -p bin/$$w-workflow-worker; \
		cd src/workers/$$w-workflow-worker; go build -o ../../../bin/$$w-workflow-worker/$$w-workflow-worker main.go; cd ../../../; \
	done

clean-workers:
	@for w in $(WORKFLOWS); do \
		/bin/rm -rf bin/$$w-workflow-worker; \
	done

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

clean: clean-workers clean-workflowclient

test:
	@echo "No tests yet"

tidy:
	@echo "No dependencies to clean"

