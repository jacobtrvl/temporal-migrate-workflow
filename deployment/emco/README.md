## Use EMCO to deploy Temporal `workers` and `workflow clients`

In this directory, you can find example `emco manifests`, which can be used to deploy sample `Temporal Worker` and sample
`Workflow Client`. 

## Prerequisites

In order to make sure that everything will work fine, you have to adjust application configuration via
profile packages (Profiles can be found at `./pkg/profiles`). Please adjust fields:
  * `temporalServer` 
  * `repository` 
  * `tag`

Also adjust `./root/emco-cfg.yaml` file to your environment.

Moreover, you have to copy Kubernetes `kube-config` files to the `./cfg` directory. 
By default, one k8s clusters should be defined:
  * `./cfg/meh1.config`

To add more clusters to the `Logical Cloud`, you can use existing manifests.
If you want to define more than 1 cluster, please adjust (uncomment cluster definitions) in the:
  * `./emco-manifests/1-prerequisites.yaml`

Define appropriate placement intent in the:

  * `./emco-manifests/3-deployment.yaml`

And create additional `kube-configs`:
  * `./cfg/core.config`, `./cfg/meh2.config`, `./cfg/ran.config`

## Deploy

To deploy sample `worker` and `workflow-client` as EMCO CompositeApplication use
`/emco-manifests/deploy.sh` script.

```bash
cd ./emco-manifests
```

```bash
./deploy.sh
```

## Verify

*Inside ./emco-manifests directory*

```bash
kubectl --kubeconfig ../cfg/meh1.config -n default get pod | grep -e worker -e workflowclient
```

## Clear Up

To clear-up an environment use `/emco-manifests/clear-up.sh` script.

*Inside ./emco-manifests directory*

```bash
./clear-up.sh
```

---

Please note, that this is not an extensive guide how to use EMCO nor Temporal.