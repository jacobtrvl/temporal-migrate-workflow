#!/bin/bash

emcoctl --config emco-cfg.yaml apply -f 1-prerequisites.yaml -v values.yaml
emcoctl --config emco-cfg.yaml apply -f 2-instantiate-lc.yaml -v values.yaml

sleep 1

emcoctl --config emco-cfg.yaml apply -f 3-deployment.yaml -v values.yaml
emcoctl --config emco-cfg.yaml apply -f 4-instantiate-dig.yaml -v values.yaml

