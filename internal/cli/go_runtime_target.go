package cli

import (
	"fmt"
	"strings"

	goruntime "github.com/panshuai/ai-sre/internal/go_runtime"
)

type diagnoseFlag struct {
	value string
	kind  goruntime.K8sResourceKind
}

func resolveDiagnoseK8sTarget(opts goRuntimeCLIOptions) (*goruntime.K8sResourceRef, error) {
	flags := []diagnoseFlag{
		{opts.PodTarget, goruntime.K8sKindPod},
		{opts.Deployment, goruntime.K8sKindDeployment},
		{opts.StatefulSet, goruntime.K8sKindStatefulSet},
		{opts.DaemonSet, goruntime.K8sKindDaemonSet},
		{opts.ReplicaSet, goruntime.K8sKindReplicaSet},
		{opts.Job, goruntime.K8sKindJob},
		{opts.CronJob, goruntime.K8sKindCronJob},
		{opts.Service, goruntime.K8sKindService},
		{opts.Ingress, goruntime.K8sKindIngress},
		{opts.PVC, goruntime.K8sKindPVC},
	}
	var picked *goruntime.K8sResourceRef
	count := 0
	for _, f := range flags {
		v := strings.TrimSpace(f.value)
		if v == "" {
			continue
		}
		count++
		ref, err := goruntime.NewK8sResourceRef(f.kind, v)
		if err != nil {
			return nil, fmt.Errorf("--%s: %w", string(f.kind), err)
		}
		picked = &ref
	}
	if count > 1 {
		return nil, fmt.Errorf("K8s 资源参数只能指定一种（--pod、--deployment、--ingress 等）")
	}
	return picked, nil
}
