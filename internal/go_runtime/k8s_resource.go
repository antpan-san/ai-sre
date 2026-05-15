package go_runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// K8sResourceKind identifies a diagnosable Kubernetes object (not Namespace/ConfigMap, etc.).
type K8sResourceKind string

const (
	K8sKindPod         K8sResourceKind = "pod"
	K8sKindDeployment  K8sResourceKind = "deployment"
	K8sKindStatefulSet K8sResourceKind = "statefulset"
	K8sKindDaemonSet   K8sResourceKind = "daemonset"
	K8sKindReplicaSet  K8sResourceKind = "replicaset"
	K8sKindJob         K8sResourceKind = "job"
	K8sKindCronJob     K8sResourceKind = "cronjob"
	K8sKindService     K8sResourceKind = "service"
	K8sKindIngress     K8sResourceKind = "ingress"
	K8sKindPVC         K8sResourceKind = "persistentvolumeclaim"
)

// K8sResourceRef is a user-selected diagnose target in the cluster.
type K8sResourceRef struct {
	Kind      K8sResourceKind
	Namespace string
	Name      string
}

func (r K8sResourceRef) String() string {
	ns := strings.TrimSpace(r.Namespace)
	name := strings.TrimSpace(r.Name)
	if ns == "" {
		return string(r.Kind) + "/" + name
	}
	return string(r.Kind) + "/" + ns + "/" + name
}

func (r K8sResourceRef) hasWorkloadPods() bool {
	switch r.Kind {
	case K8sKindPod, K8sKindDeployment, K8sKindStatefulSet, K8sKindDaemonSet, K8sKindReplicaSet, K8sKindJob, K8sKindCronJob:
		return true
	default:
		return false
	}
}

// ParseNamespacedTarget parses name or namespace/name.
func ParseNamespacedTarget(target string) (namespace, name string, err error) {
	target = strings.Trim(strings.TrimSpace(target), "/")
	if target == "" {
		return "", "", fmt.Errorf("资源名称不能为空")
	}
	parts := strings.Split(target, "/")
	switch len(parts) {
	case 1:
		return "", parts[0], nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("格式应为 name 或 namespace/name")
	}
}

func NewK8sResourceRef(kind K8sResourceKind, target string) (K8sResourceRef, error) {
	ns, name, err := ParseNamespacedTarget(target)
	if err != nil {
		return K8sResourceRef{}, err
	}
	return K8sResourceRef{Kind: kind, Namespace: ns, Name: name}, nil
}

// CollectKubernetesDiagnose runs proc watch (when applicable) plus resource-level kubectl probes.
func CollectKubernetesDiagnose(ctx context.Context, ref K8sResourceRef, opts KubernetesCollectOptions, interval time.Duration, count int) (*WatchReport, error) {
	k := kubectlBin(opts.KubectlPath)
	if ref.Namespace == "" {
		ref.Namespace = currentKubectlNamespace(ctx, k)
	}
	if ref.Kind == K8sKindPod {
		opts.Target = formatPodTarget(ref)
		wr, err := CollectKubernetesWatch(ctx, opts, interval, count)
		if wr != nil {
			stampK8sResourceTarget(wr, ref)
		}
		return wr, err
	}
	resBundle := GatherK8sResourceProbeBundle(ctx, k, ref)
	if ref.hasWorkloadPods() {
		podRef, note, err := ResolvePrimaryPod(ctx, k, ref)
		if err != nil {
			wr := buildResourceOnlyReport(ref, resBundle, err)
			return wr, nil
		}
		podOpts := opts
		podOpts.Target = formatPodTargetFromRef(podRef)
		wr, err := CollectKubernetesWatch(ctx, podOpts, interval, count)
		if wr == nil {
			wr = buildResourceOnlyReport(ref, resBundle, err)
		} else if err != nil {
			wr.Errors = append(wr.Errors, err.Error())
		}
		if note != "" {
			wr.Errors = append(wr.Errors, note)
		}
		stampK8sResourceTarget(wr, ref)
		attachProbeBundle(wr, resBundle)
		if wr.Target.Pod == "" {
			wr.Target.Pod = podRef.Pod
			wr.Target.Namespace = podRef.Namespace
			wr.Target.Container = podRef.Container
		}
		return wr, nil
	}
	wr := buildResourceOnlyReport(ref, resBundle, nil)
	return wr, nil
}

func formatPodTarget(ref K8sResourceRef) string {
	ns, name := ref.Namespace, ref.Name
	if ns == "" {
		return name
	}
	return ns + "/" + name
}

func formatPodTargetFromRef(pod PodRef) string {
	if pod.Namespace == "" {
		return pod.Pod
	}
	if pod.Container != "" {
		return pod.Namespace + "/" + pod.Pod + "/" + pod.Container
	}
	return pod.Namespace + "/" + pod.Pod
}

func stampK8sResourceTarget(wr *WatchReport, ref K8sResourceRef) {
	if wr == nil {
		return
	}
	wr.Target.Source = "kubernetes"
	wr.Target.ResourceKind = string(ref.Kind)
	wr.Target.ResourceName = ref.Name
	if wr.Target.Namespace == "" {
		wr.Target.Namespace = ref.Namespace
	}
	if wr.Target.Target == "" {
		wr.Target.Target = ref.String()
	}
}

func buildResourceOnlyReport(ref K8sResourceRef, bundle map[string]string, probeErr error) *WatchReport {
	wr := &WatchReport{
		GeneratedAt: time.Now(),
		Target: ProcessIdentity{
			Namespace:    ref.Namespace,
			ResourceKind: string(ref.Kind),
			ResourceName: ref.Name,
			Target:       ref.String(),
			Source:       "kubernetes",
		},
		SampleCount: 0,
		ProbeBundle: bundle,
	}
	attachProbeBundle(wr, bundle)
	findings := analyzeResourceProbeBundle(ref, bundle, probeErr)
	wr.TrendFindings = findings
	if probeErr != nil {
		wr.Errors = append(wr.Errors, probeErr.Error())
	}
	wr.Summary = SummarizeInfrastructureReport(wr)
	return wr
}

func analyzeResourceProbeBundle(ref K8sResourceRef, bundle map[string]string, err error) []Finding {
	var out []Finding
	if err != nil {
		out = append(out, Finding{
			Severity: severityCrit,
			Title:    fmt.Sprintf("无法解析 %s 关联 Pod", ref.Kind),
			Evidence: err.Error(),
			Cause:    "资源不存在、无就绪 Pod 或 kubectl 无法访问集群",
		})
	}
	text := strings.ToLower(probeBundleText(bundle, 48_000))
	switch ref.Kind {
	case K8sKindIngress:
		if strings.Contains(text, "no endpoints") || strings.Contains(text, "endpoints not found") {
			out = append(out, Finding{
				Severity: severityCrit,
				Title:    "Ingress 后端无可用 Endpoints",
				Evidence: probeSnippet(bundle, "kubectl_ingress_endpoints", 800),
				Cause:    "Service 无就绪 Pod 或 selector 不匹配",
			})
		}
		if strings.Contains(text, "certificate") || strings.Contains(text, "tls") {
			out = append(out, Finding{
				Severity: severityWarn,
				Title:    "Ingress TLS/证书相关异常",
				Evidence: probeSnippet(bundle, "kubectl_ingress_describe", 600),
				Cause:    "Secret 缺失、证书过期或 issuer 配置错误",
			})
		}
	case K8sKindService:
		if strings.Contains(text, "<none>") && strings.Contains(text, "endpoints") {
			out = append(out, Finding{
				Severity: severityCrit,
				Title:    "Service 无后端 Endpoints",
				Evidence: probeSnippet(bundle, "kubectl_service_endpoints", 800),
				Cause:    "无匹配 Pod 或 Pod 未就绪",
			})
		}
	case K8sKindPVC:
		if strings.Contains(text, "pending") {
			out = append(out, Finding{
				Severity: severityWarn,
				Title:    "PVC 处于 Pending",
				Evidence: probeSnippet(bundle, "kubectl_pvc_describe", 800),
				Cause:    "存储类、配额或调度器无法绑定卷",
			})
		}
	}
	return out
}

// ResolvePrimaryPod picks the most problematic Pod owned by a workload.
func ResolvePrimaryPod(ctx context.Context, kubectl string, ref K8sResourceRef) (PodRef, string, error) {
	kubectl = kubectlBin(kubectl)
	ns := ref.Namespace
	if ns == "" {
		ns = currentKubectlNamespace(ctx, kubectl)
		ref.Namespace = ns
	}
	var pods []podListItem
	var note string
	var err error
	switch ref.Kind {
	case K8sKindDeployment:
		pods, err = podsForSelector(ctx, kubectl, ns, deploySelector(ctx, kubectl, ns, ref.Name))
	case K8sKindStatefulSet:
		pods, err = podsForSelector(ctx, kubectl, ns, statefulSetSelector(ctx, kubectl, ns, ref.Name))
	case K8sKindDaemonSet:
		pods, err = podsForSelector(ctx, kubectl, ns, daemonSetSelector(ctx, kubectl, ns, ref.Name))
	case K8sKindReplicaSet:
		pods, err = podsForSelector(ctx, kubectl, ns, replicaSetSelector(ctx, kubectl, ns, ref.Name))
	case K8sKindJob:
		pods, err = podsLabeled(ctx, kubectl, ns, "job-name="+ref.Name)
	case K8sKindCronJob:
		pods, note, err = podsForCronJob(ctx, kubectl, ns, ref.Name)
	default:
		return PodRef{}, "", fmt.Errorf("unsupported workload kind %q", ref.Kind)
	}
	if err != nil {
		return PodRef{}, note, err
	}
	if len(pods) == 0 {
		return PodRef{}, note, fmt.Errorf("%s/%s 下没有 Pod", ns, ref.Name)
	}
	sort.Slice(pods, func(i, j int) bool {
		return podRank(pods[i]) > podRank(pods[j])
	})
	p := pods[0]
	return PodRef{Namespace: p.Namespace, Pod: p.Name}, note, nil
}

type podListItem struct {
	Namespace     string
	Name          string
	Phase         string
	Ready         bool
	RestartCount  int32
	WaitingReason string
}

func podRank(p podListItem) int {
	score := 0
	if p.Phase != "Running" {
		score += 100
	}
	if !p.Ready {
		score += 50
	}
	if p.RestartCount > 0 {
		score += int(min32(p.RestartCount, 20)) * 5
	}
	switch p.WaitingReason {
	case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "CreateContainerConfigError":
		score += 80
	}
	return score
}

func min32(a int32, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func podsForCronJob(ctx context.Context, kubectl, ns, name string) ([]podListItem, string, error) {
	jobName, err := latestJobForCronJob(ctx, kubectl, ns, name)
	if err != nil {
		return nil, "", err
	}
	note := "使用最近 Job " + jobName
	pods, err := podsLabeled(ctx, kubectl, ns, "job-name="+jobName)
	return pods, note, err
}

func latestJobForCronJob(ctx context.Context, kubectl, ns, cronName string) (string, error) {
	out, err := kubectlOutput(ctx, kubectl, 20*time.Second, "get", "jobs", "-n", ns,
		"-l", "cronjob.kubernetes.io/cronjob-name="+cronName,
		"--sort-by=.metadata.creationTimestamp",
		"-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if n := strings.TrimSpace(lines[i]); n != "" {
			return n, nil
		}
	}
	return "", fmt.Errorf("CronJob %s/%s 没有关联 Job", ns, cronName)
}

func deploySelector(ctx context.Context, kubectl, ns, name string) map[string]string {
	return selectorFromResource(ctx, kubectl, ns, "deployment", name)
}

func statefulSetSelector(ctx context.Context, kubectl, ns, name string) map[string]string {
	return selectorFromResource(ctx, kubectl, ns, "statefulset", name)
}

func daemonSetSelector(ctx context.Context, kubectl, ns, name string) map[string]string {
	return selectorFromResource(ctx, kubectl, ns, "daemonset", name)
}

func replicaSetSelector(ctx context.Context, kubectl, ns, name string) map[string]string {
	return selectorFromResource(ctx, kubectl, ns, "replicaset", name)
}

func selectorFromResource(ctx context.Context, kubectl, ns, resource, name string) map[string]string {
	out, err := kubectlOutput(ctx, kubectl, 15*time.Second, "get", resource, name, "-n", ns, "-o", "json")
	if err != nil {
		return nil
	}
	var doc struct {
		Spec struct {
			Selector struct {
				MatchLabels map[string]string `json:"matchLabels"`
			} `json:"selector"`
		} `json:"spec"`
	}
	if json.Unmarshal([]byte(out), &doc) != nil {
		return nil
	}
	return doc.Spec.Selector.MatchLabels
}

func podsForSelector(ctx context.Context, kubectl, ns string, labels map[string]string) ([]podListItem, error) {
	if len(labels) == 0 {
		return nil, fmt.Errorf("资源无 selector，无法列出 Pod")
	}
	var parts []string
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return podsLabeled(ctx, kubectl, ns, strings.Join(parts, ","))
}

func podsLabeled(ctx context.Context, kubectl, ns, labelSelector string) ([]podListItem, error) {
	out, err := kubectlOutput(ctx, kubectl, 25*time.Second, "get", "pods", "-n", ns, "-l", labelSelector, "-o", "json")
	if err != nil {
		return nil, err
	}
	var pl struct {
		Items []kubectlPodDetail `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &pl); err != nil {
		return nil, err
	}
	var pods []podListItem
	for _, it := range pl.Items {
		p := podListItem{
			Namespace: it.Metadata.Namespace,
			Name:      it.Metadata.Name,
			Phase:     it.Status.Phase,
		}
		if p.Namespace == "" {
			p.Namespace = ns
		}
		for _, c := range it.Status.ContainerStatuses {
			if c.Ready {
				p.Ready = true
			}
			if c.RestartCount > p.RestartCount {
				p.RestartCount = c.RestartCount
			}
			if c.State.Waiting != nil && p.WaitingReason == "" {
				p.WaitingReason = c.State.Waiting.Reason
			}
		}
		pods = append(pods, p)
	}
	return pods, nil
}
