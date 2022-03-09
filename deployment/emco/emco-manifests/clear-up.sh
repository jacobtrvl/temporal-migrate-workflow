#!/bin/bash

emcoctl --config emco-cfg.yaml delete -f 4-instantiate-dig.yaml -v values.yaml

sleep 6

emcoctl --config emco-cfg.yaml delete -f 3-deployment.yaml -v values.yaml

sleep 2

emcoctl --config emco-cfg.yaml delete -f 2-instantiate-lc.yaml -v values.yaml

sleep 2

emcoctl --config emco-cfg.yaml delete -f 1-prerequisites.yaml -v values.yaml

