module gitlab.com/project-emco/samples/temporal/migrate-workflow

go 1.16

require (
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	gitlab.com/project-emco/core/emco-base/src/workflowmgr v0.0.0-00010101000000-000000000000
	go.temporal.io/sdk v1.13.1
)

replace (
	gitlab.com/project-emco/core/emco-base => ../emco-base
	gitlab.com/project-emco/core/emco-base/src/clm => ../emco-base/src/clm
	gitlab.com/project-emco/core/emco-base/src/monitor => ../emco-base/src/monitor
	gitlab.com/project-emco/core/emco-base/src/orchestrator => ../emco-base/src/orchestrator
	gitlab.com/project-emco/core/emco-base/src/rsync => ../emco-base/src/rsync
	gitlab.com/project-emco/core/emco-base/src/workflowmgr => ../emco-base/src/workflowmgr
	gitlab.com/project-emco/core/emco-base/src/workflowmgr/pkg/emcotemporalapi => ../emco-base/src/workflowmgr/pkg/emcotemporalapi
)
