package go_runtime

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// GatherK8sResourceProbeBundle collects describe/events/endpoints for a K8s resource.
func GatherK8sResourceProbeBundle(ctx context.Context, kubectl string, ref K8sResourceRef) map[string]string {
	kubectl = kubectlBin(kubectl)
	ns := ref.Namespace
	if ns == "" {
		ns = currentKubectlNamespace(ctx, kubectl)
	}
	name := ref.Name
	out := map[string]string{}
	put := func(k, v string) {
		v = truncateProbe(v, maxProbeBytes)
		if strings.TrimSpace(v) != "" {
			out[k] = v
		}
	}
	put("k8s_resource_ref", ref.String())
	switch ref.Kind {
	case K8sKindDeployment:
		put("kubectl_deployment_describe", kubectlOutputSafe(ctx, kubectl, 30*time.Second, "describe", "deployment", name, "-n", ns))
		put("kubectl_deployment_yaml", kubectlOutputSafe(ctx, kubectl, 15*time.Second, "get", "deployment", name, "-n", ns, "-o", "yaml"))
	case K8sKindStatefulSet:
		put("kubectl_statefulset_describe", kubectlOutputSafe(ctx, kubectl, 30*time.Second, "describe", "statefulset", name, "-n", ns))
	case K8sKindDaemonSet:
		put("kubectl_daemonset_describe", kubectlOutputSafe(ctx, kubectl, 30*time.Second, "describe", "daemonset", name, "-n", ns))
	case K8sKindReplicaSet:
		put("kubectl_replicaset_describe", kubectlOutputSafe(ctx, kubectl, 30*time.Second, "describe", "replicaset", name, "-n", ns))
	case K8sKindJob:
		put("kubectl_job_describe", kubectlOutputSafe(ctx, kubectl, 25*time.Second, "describe", "job", name, "-n", ns))
	case K8sKindCronJob:
		put("kubectl_cronjob_describe", kubectlOutputSafe(ctx, kubectl, 25*time.Second, "describe", "cronjob", name, "-n", ns))
	case K8sKindService:
		put("kubectl_service_describe", kubectlOutputSafe(ctx, kubectl, 25*time.Second, "describe", "service", name, "-n", ns))
		put("kubectl_service_endpoints", kubectlOutputSafe(ctx, kubectl, 20*time.Second, "get", "endpoints", name, "-n", ns, "-o", "yaml"))
		gatherServiceBackendPods(ctx, kubectl, ns, name, put)
	case K8sKindIngress:
		put("kubectl_ingress_describe", kubectlOutputSafe(ctx, kubectl, 30*time.Second, "describe", "ingress", name, "-n", ns))
		put("kubectl_ingress_yaml", kubectlOutputSafe(ctx, kubectl, 20*time.Second, "get", "ingress", name, "-n", ns, "-o", "yaml"))
		gatherIngressBackends(ctx, kubectl, ns, name, put)
	case K8sKindPVC:
		put("kubectl_pvc_describe", kubectlOutputSafe(ctx, kubectl, 25*time.Second, "describe", "pvc", name, "-n", ns))
		put("kubectl_pvc_yaml", kubectlOutputSafe(ctx, kubectl, 15*time.Second, "get", "pvc", name, "-n", ns, "-o", "yaml"))
	case K8sKindPod:
		return GatherPodProbeBundle(ctx, kubectl, ns, name)
	}
	ev := kubectlOutputSafe(ctx, kubectl, 18*time.Second, "get", "events", "-n", ns,
		"--field-selector=involvedObject.name="+name, "-o", "wide")
	put("kubectl_resource_events", ev)
	if len(out) == 0 {
		return nil
	}
	return out
}

func gatherServiceBackendPods(ctx context.Context, kubectl, ns, svc string, put func(string, string)) {
	sel := kubectlOutputSafe(ctx, kubectl, 12*time.Second, "get", "service", svc, "-n", ns, "-o", "jsonpath={.spec.selector}")
	sel = strings.TrimSpace(sel)
	if sel == "" || sel == "map[]" {
		return
	}
	// jsonpath map format: map[key:value ...] — fallback to json
	out, _ := kubectlOutput(ctx, kubectl, 12*time.Second, "get", "service", svc, "-n", ns, "-o", "json")
	var doc struct {
		Spec struct {
			Selector map[string]string `json:"selector"`
		} `json:"spec"`
	}
	if json.Unmarshal([]byte(out), &doc) == nil && len(doc.Spec.Selector) > 0 {
		pods, err := podsForSelector(ctx, kubectl, ns, doc.Spec.Selector)
		if err != nil || len(pods) == 0 {
			return
		}
		p := pods[0]
		if len(pods) > 1 {
			sort.Slice(pods, func(i, j int) bool { return podRank(pods[i]) > podRank(pods[j]) })
			p = pods[0]
		}
		b := GatherPodProbeBundle(ctx, kubectl, p.Namespace, p.Name)
		for k, v := range b {
			put("kubectl_service_pod_"+k, v)
		}
	}
}

func gatherIngressBackends(ctx context.Context, kubectl, ns, ing string, put func(string, string)) {
	svcs := kubectlOutputSafe(ctx, kubectl, 15*time.Second, "get", "ingress", ing, "-n", ns,
		"-o", "jsonpath={range .spec.rules[*]}{range .http.paths[*]}{.backend.service.name}{\"\\n\"}{end}{end}")
	seen := map[string]struct{}{}
	for _, svc := range strings.Split(svcs, "\n") {
		svc = strings.TrimSpace(svc)
		if svc == "" {
			continue
		}
		if _, ok := seen[svc]; ok {
			continue
		}
		seen[svc] = struct{}{}
		put("kubectl_ingress_service_"+svc+"_endpoints",
			kubectlOutputSafe(ctx, kubectl, 15*time.Second, "get", "endpoints", svc, "-n", ns, "-o", "yaml"))
		desc := kubectlOutputSafe(ctx, kubectl, 20*time.Second, "describe", "service", svc, "-n", ns)
		put("kubectl_ingress_service_"+svc+"_describe", desc)
	}
	var svcNames []string
	for svc := range seen {
		svcNames = append(svcNames, svc)
	}
	sort.Strings(svcNames)
	put("kubectl_ingress_backend_services", strings.Join(svcNames, ", "))
}
