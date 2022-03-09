## Deployment

To deploy sample application you have to move to `emco-manifests` directory

```bash
$ cd ./emco-manifests
```

### 1. Deploy sample application

Prepare environment and deploy sample application. These operations require that you
have EMCO installed in your environment.

```bash
$ emcoctl --config emco-cfg.yaml -v values.yaml apply -f 00.define-clusters-proj.yaml
```

```bash
$ emcoctl --config emco-cfg.yaml -v values.yaml apply -f 01.instantiate-lc.yaml
```

```bash
$ emcoctl --config emco-cfg.yaml -v values.yaml apply -f 02.define-app-dig.yaml
```

```bash
$ emcoctl --config emco-cfg.yaml -v values.yaml apply -f 03.instantiate-dig.yaml
```

### 2. Deploy sample workflow

This operation requires that `workflowmgr` microservice is installed with EMCO. 
Moreover, please remember to adjust `04.define-workflow-1.yaml` file that it's 
compatible with your environment.

```bash
$ emcoctl --config emco-cfg.yaml -v values.yaml apply -f 04.define-workflow-1.yaml
```

### 3. Start sample workflow

Please notice that you have to deploy `Temporal Worker` and `Temporal Workflow Client`
before workflow will be executed. Moreover, you have to deploy `Temporal Server`.

```bash
$ emcoctl --config emco-cfg.yaml -v values.yaml delete -f 05.start-workflow.yaml
```

### 4. Other operations

```bash
$ emcoctl --config emco-cfg.yaml -v values.yaml delete -f 06.get-workflow-status.yaml 
$ emcoctl --config emco-cfg.yaml -v values.yaml delete -f 07.cancel-workflow.yaml 
```