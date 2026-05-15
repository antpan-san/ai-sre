package go_runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type kubectlPodDetail struct {
	kubectlPodJSON
	Status struct {
		Phase string `json:"phase"`
		Conditions []struct {
			Type    string `json:"type"`
			Status  string `json:"status"`
			Reason  string `json:"reason"`
			Message string `json:"message"`
		} `json:"conditions"`
		ContainerStatuses []struct {
			Name         string `json:"name"`
			Ready        bool   `json:"ready"`
			RestartCount int32  `json:"restartCount"`
			ContainerID  string `json:"containerID"`
			State        struct {
				Running    *struct{ StartedAt string `json:"startedAt"` } `json:"running"`
				Waiting    *struct{ Reason, Message string }            `json:"waiting"`
				Terminated *struct {
					Reason   string `json:"reason"`
					Message  string `json:"message"`
					ExitCode int32  `json:"exitCode"`
				} `json:"terminated"`
			} `json:"state"`
			LastState struct {
				Terminated *struct {
					Reason   string `json:"reason"`
					Message  string `json:"message"`
					ExitCode int32  `json:"exitCode"`
				} `json:"terminated"`
			} `json:"lastState"`
		} `json:"containerStatuses"`
	} `json:"status"`
	Spec struct {
		NodeName string `json:"nodeName"`
		Containers []struct {
			Name string `json:"name"`
			Resources struct {
				Limits struct {
					Memory string `json:"memory"`
					CPU    string `json:"cpu"`
				} `json:"limits"`
			} `json:"resources"`
		} `json:"containers"`
	} `json:"spec"`
}

func loadPodDetail(ctx context.Context, kubectl, ns, pod string) (kubectlPodDetail, error) {
	out, err := kubectlOutput(ctx, kubectl, 20*time.Second, "get", "pod", "-n", ns, pod, "-o", "json")
	if err != nil {
		return kubectlPodDetail{}, err
	}
	var parsed kubectlPodDetail
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return kubectlPodDetail{}, err
	}
	return parsed, nil
}

func collectorImageCandidates(opts KubernetesCollectOptions, nodeImages []string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(img string) {
		img = strings.TrimSpace(img)
		if img == "" {
			return
		}
		if _, ok := seen[img]; ok {
			return
		}
		seen[img] = struct{}{}
		out = append(out, img)
	}
	add(opts.CollectorImage)
	for _, img := range nodeImages {
		add(img)
	}
	for _, img := range []string{
		"busybox:1.36",
		"docker.io/library/busybox:1.36",
	} {
		add(img)
	}
	return out
}

func imagesOnNode(ctx context.Context, kubectl, node string) []string {
	if strings.TrimSpace(node) == "" {
		return nil
	}
	out, err := kubectlOutput(ctx, kubectl, 25*time.Second,
		"get", "pods", "-A",
		"--field-selector=spec.nodeName="+node,
		"-o", "jsonpath={range .items[*]}{range .spec.containers[*]}{.image}{\"\\n\"}{end}{end}",
	)
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	var imgs []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		low := strings.ToLower(line)
		if !strings.Contains(low, "busybox") && !strings.Contains(low, "alpine") && !strings.Contains(low, "pause") {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		imgs = append(imgs, line)
	}
	return imgs
}

func ensureCollectorPodWithFallback(ctx context.Context, kubectl, ns, name, node string, images []string) (string, error) {
	var lastErr error
	for _, image := range images {
		deleteCollectorPod(ctx, kubectl, ns, name)
		if err := applyCollectorPod(ctx, kubectl, ns, name, node, image); err != nil {
			lastErr = err
			continue
		}
		if err := waitCollectorReady(ctx, kubectl, ns, name); err == nil {
			return image, nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no collector image candidates")
	}
	return "", lastErr
}

func applyCollectorPod(ctx context.Context, kubectl, ns, name, node, image string) error {
	manifest := collectorPodManifest(name, ns, node, image)
	_, err := kubectlInput(ctx, kubectl, 30*time.Second, manifest, "apply", "-f", "-")
	return err
}

func collectorPodManifest(name, ns, node, image string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: ai-sre-go-runtime-diagnose
spec:
  restartPolicy: Never
  nodeName: %s
  tolerations:
  - operator: Exists
  containers:
  - name: collector
    image: %s
    command: ["sh", "-c", "sleep 3600"]
    securityContext:
      privileged: true
      runAsUser: 0
    volumeMounts:
    - name: host-proc
      mountPath: /host/proc
      readOnly: true
    - name: host-cgroup
      mountPath: /host/sys/fs/cgroup
      readOnly: true
  volumes:
  - name: host-proc
    hostPath:
      path: /proc
      type: Directory
  - name: host-cgroup
    hostPath:
      path: /sys/fs/cgroup
      type: Directory
`, name, ns, node, image)
}

func waitCollectorReady(ctx context.Context, kubectl, ns, name string) error {
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		phase, reason, msg, err := collectorPodPhase(ctx, kubectl, ns, name)
		if err != nil {
			return err
		}
		switch phase {
		case "Running", "Succeeded":
			return nil
		case "Failed":
			return fmt.Errorf("collector pod failed: %s %s", reason, msg)
		case "Pending", "Unknown":
			if reason == "ImagePullBackOff" || reason == "ErrImagePull" || reason == "InvalidImageName" {
				return fmt.Errorf("collector image pull failed: %s (%s)", reason, msg)
			}
		}
		time.Sleep(2 * time.Second)
	}
	_, reason, msg, _ := collectorPodPhase(ctx, kubectl, ns, name)
	if reason != "" {
		return fmt.Errorf("collector not ready: %s (%s)", reason, msg)
	}
	return fmt.Errorf("timed out waiting for collector pod/%s to become Ready", name)
}

func collectorPodPhase(ctx context.Context, kubectl, ns, name string) (phase, reason, message string, err error) {
	out, err := kubectlOutput(ctx, kubectl, 12*time.Second, "-n", ns, "get", "pod", name,
		"-o", "jsonpath={.status.phase}{\"\\t\"}{.status.containerStatuses[0].state.waiting.reason}{\"\\t\"}{.status.containerStatuses[0].state.waiting.message}")
	if err != nil {
		return "", "", "", err
	}
	parts := strings.Split(strings.TrimSpace(out), "\t")
	phase = parts[0]
	if len(parts) > 1 {
		reason = parts[1]
	}
	if len(parts) > 2 {
		message = parts[2]
	}
	return phase, reason, message, nil
}

func describePodEvents(ctx context.Context, kubectl, ns, name string) string {
	out, err := kubectlOutput(ctx, kubectl, 15*time.Second, "describe", "pod", "-n", ns, name)
	if err != nil {
		return err.Error()
	}
	lines := strings.Split(out, "\n")
	var events []string
	inEvents := false
	for _, line := range lines {
		if strings.HasPrefix(line, "Events:") {
			inEvents = true
			continue
		}
		if inEvents {
			if strings.TrimSpace(line) == "" && len(events) > 0 {
				break
			}
			events = append(events, line)
		}
	}
	if len(events) == 0 {
		return strings.TrimSpace(out)
	}
	return strings.Join(events, "\n")
}

func analyzeTargetPodDetail(d kubectlPodDetail, ref PodRef) []Finding {
	var out []Finding
	phase := strings.TrimSpace(d.Status.Phase)
	if phase != "" && phase != "Running" {
		out = append(out, Finding{
			Severity: severityWarn,
			Title:    "目标 Pod 未处于 Running",
			Evidence: fmt.Sprintf("phase=%s node=%s", phase, d.Spec.NodeName),
			Cause:    "业务容器可能尚未就绪、正在重启或调度失败",
		})
	}
	for _, c := range d.Status.ContainerStatuses {
		if c.RestartCount > 0 {
			sev := severityWarn
			if c.RestartCount >= 3 {
				sev = severityCrit
			}
			out = append(out, Finding{
				Severity: sev,
				Title:    fmt.Sprintf("容器 %s 已重启 %d 次", c.Name, c.RestartCount),
				Evidence: containerStateEvidence(c),
				Cause:    "进程崩溃、OOM、探针失败或主动重建可能导致重启",
			})
		}
		if c.State.Waiting != nil && c.State.Waiting.Reason != "" {
			reason := strings.TrimSpace(c.State.Waiting.Reason)
			sev, cause := waitingReasonDiagnosis(reason)
			out = append(out, Finding{
				Severity: sev,
				Title:    fmt.Sprintf("容器 %s 处于 Waiting（%s）", c.Name, reason),
				Evidence: fmt.Sprintf("reason=%s message=%s", reason, c.State.Waiting.Message),
				Cause:    cause,
			})
		}
		if c.LastState.Terminated != nil && c.LastState.Terminated.Reason == "OOMKilled" {
			out = append(out, Finding{
				Severity: severityCrit,
				Title:    fmt.Sprintf("容器 %s 曾因 OOM 退出", c.Name),
				Evidence: fmt.Sprintf("exitCode=%d message=%s", c.LastState.Terminated.ExitCode, c.LastState.Terminated.Message),
				Cause:    "内存超过 limit 或节点内存压力",
				Verify:   "检查 memory limit 与进程堆增长",
			})
		}
	}
	for _, c := range d.Spec.Containers {
		if strings.TrimSpace(c.Resources.Limits.Memory) != "" {
			out = append(out, Finding{
				Severity: severityInfo,
				Title:    fmt.Sprintf("容器 %s 配置了 memory limit", c.Name),
				Evidence: fmt.Sprintf("memory.limit=%s", c.Resources.Limits.Memory),
				Cause:    "超过 limit 会触发 OOMKill",
				Verify:   "结合运行时 RSS 趋势判断是否触顶",
			})
		}
	}
	return out
}

func waitingReasonDiagnosis(reason string) (severity, cause string) {
	severity, cause = severityWarn, "容器启动被阻塞，需结合 describe/events 与日志确认"
	switch reason {
	case "ImagePullBackOff", "ErrImagePull", "InvalidImageName":
		return severityCrit, "业务容器镜像无法拉取：检查 image/tag、仓库可达性、imagePullPolicy 及节点是否已有本地镜像"
	case "CrashLoopBackOff":
		return severityCrit, "业务容器反复崩溃退出（常见 exit≠0、OOMKilled、启动命令错误）；查看 logs --previous 与 LastState"
	case "CreateContainerConfigError":
		return severityCrit, "容器配置无效：启动命令、挂载、环境变量或 Secret/ConfigMap 引用错误"
	case "CreateContainerError":
		return severityCrit, "容器创建失败：镜像损坏、架构不匹配或运行时拒绝创建"
	case "RunContainerError":
		return severityCrit, "容器启动后立即失败：可执行文件不存在或权限不足"
	default:
		return severity, cause
	}
}

func containerStateEvidence(c struct {
	Name         string `json:"name"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	ContainerID  string `json:"containerID"`
	State        struct {
		Running    *struct{ StartedAt string `json:"startedAt"` } `json:"running"`
		Waiting    *struct{ Reason, Message string }            `json:"waiting"`
		Terminated *struct {
			Reason   string `json:"reason"`
			Message  string `json:"message"`
			ExitCode int32  `json:"exitCode"`
		} `json:"terminated"`
	} `json:"state"`
	LastState struct {
		Terminated *struct {
			Reason   string `json:"reason"`
			Message  string `json:"message"`
			ExitCode int32  `json:"exitCode"`
		} `json:"terminated"`
	} `json:"lastState"`
}) string {
	parts := []string{fmt.Sprintf("ready=%v", c.Ready), fmt.Sprintf("restarts=%d", c.RestartCount)}
	if c.ContainerID != "" {
		parts = append(parts, "id="+shortID(c.ContainerID))
	}
	return strings.Join(parts, " ")
}

func analyzeCollectorFailure(collectorName, collectorNS, image string, err error, events string) []Finding {
	reason := err.Error()
	var out []Finding
	out = append(out, Finding{
		Severity: severityCrit,
		Title:    "诊断采集器未能启动",
		Evidence: fmt.Sprintf("collector=%s/%s image=%s err=%s", collectorNS, collectorName, image, reason),
		Cause:    "常见为节点无法拉取 busybox 镜像、禁止特权 Pod，或镜像仓库不可达",
	})
	if strings.Contains(reason, "ImagePull") || strings.Contains(events, "ImagePull") || strings.Contains(events, "ErrImagePull") {
		out = append(out, Finding{
			Severity: severityCrit,
			Title:    "采集器镜像拉取失败",
			Evidence: truncateLines(events, 6),
			Cause:    "离线/内网集群需配置可拉取的镜像",
		})
	}
	if strings.Contains(events, "privileged") || strings.Contains(strings.ToLower(events), "denied") {
		out = append(out, Finding{
			Severity: severityWarn,
			Title:    "采集器可能受安全策略限制",
			Evidence: truncateLines(events, 4),
			Cause:    "kube-system 可能禁止 privileged Pod",
		})
	}
	out = append(out, Finding{
		Severity: severityInfo,
		Title:    "已基于 Kubernetes 状态完成部分诊断",
		Evidence: "未读取宿主机 /proc（采集器未就绪）",
		Cause:    "运行时内存/FD 趋势需采集器或节点本地 PID",
	})
	return out
}

func truncateLines(s string, max int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) <= max {
		return strings.TrimSpace(s)
	}
	return strings.Join(lines[:max], "\n") + "\n..."
}

// BuildInfrastructureWatchReport returns a diagnostic report when proc collection is unavailable.
func BuildInfrastructureWatchReport(ref PodRef, detail kubectlPodDetail, containerID, collectorName, collectorNS, imageTried string, collectorErr error, collectorEvents string) *WatchReport {
	findings := analyzeTargetPodDetail(detail, ref)
	if collectorErr != nil {
		findings = append(findings, analyzeCollectorFailure(collectorName, collectorNS, imageTried, collectorErr, collectorEvents)...)
	}
	wr := &WatchReport{
		GeneratedAt: time.Now(),
		Target: ProcessIdentity{
			Namespace:   ref.Namespace,
			Pod:         ref.Pod,
			Container:   ref.Container,
			Node:        detail.Spec.NodeName,
			ContainerID: containerID,
			Target:      ref.Namespace + "/" + ref.Pod + "/" + ref.Container,
			Source:      "kubernetes",
		},
		SampleCount:   0,
		TrendFindings: findings,
		Errors:        []string{},
	}
	if collectorErr != nil {
		wr.Errors = append(wr.Errors, collectorErr.Error())
	}
	wr.Summary = SummarizeInfrastructureReport(wr)
	return wr
}

func SummarizeInfrastructureReport(wr *WatchReport) ReportSummary {
	if wr == nil {
		return ReportSummary{Level: "UNKNOWN", Title: "无诊断数据"}
	}
	top := topFinding(wr.TrendFindings)
	level := "WARN"
	title := "未能完成宿主机 proc 采集，已输出 Kubernetes 侧诊断"
	evidence := "sample_count=0"
	if top != nil {
		if strings.TrimSpace(top.Cause) != "" {
			title = top.Cause
		} else {
			title = top.Title
		}
		if top.Evidence != "" {
			evidence = top.Evidence
		}
		switch strings.ToLower(top.Severity) {
		case severityCrit:
			level = "CRITICAL"
		case severityWarn:
			level = "WARN"
		default:
			level = "INFO"
		}
	}
	return ReportSummary{
		Level:    level,
		Title:    title,
		Evidence: evidence,
	}
}
