package go_runtime

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type KubernetesCollectOptions struct {
	Target             string
	KubectlPath        string
	CollectorImage     string
	CollectorNamespace string
	KeepCollector      bool
}

type PodRef struct {
	Namespace string
	Pod       string
	Container string
}

type kubectlPodJSON struct {
	Metadata struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		NodeName string `json:"nodeName"`
	} `json:"spec"`
	Status struct {
		ContainerStatuses []struct {
			Name        string `json:"name"`
			ContainerID string `json:"containerID"`
		} `json:"containerStatuses"`
	} `json:"status"`
}

func ParsePodTarget(target string) (PodRef, error) {
	target = strings.Trim(strings.TrimSpace(target), "/")
	if target == "" {
		return PodRef{}, fmt.Errorf("pod target is required")
	}
	parts := strings.Split(target, "/")
	switch len(parts) {
	case 1:
		return PodRef{Pod: parts[0]}, nil
	case 2:
		return PodRef{Namespace: parts[0], Pod: parts[1]}, nil
	case 3:
		return PodRef{Namespace: parts[0], Pod: parts[1], Container: parts[2]}, nil
	default:
		return PodRef{}, fmt.Errorf("pod target must be pod, namespace/pod, or namespace/pod/container")
	}
}

func CollectKubernetesWatch(ctx context.Context, opts KubernetesCollectOptions, interval time.Duration, count int) (*WatchReport, error) {
	ref, err := ParsePodTarget(opts.Target)
	if err != nil {
		return nil, err
	}
	k := kubectlBin(opts.KubectlPath)
	if ref.Namespace == "" {
		ref.Namespace = currentKubectlNamespace(ctx, k)
	}
	pod, err := loadPod(ctx, k, ref.Namespace, ref.Pod)
	if err != nil && ref.Namespace != "" {
		pod, err = loadPodAcrossNamespaces(ctx, k, ref.Pod)
	}
	if err != nil {
		return nil, err
	}
	ref.Namespace = pod.Metadata.Namespace
	ref.Pod = pod.Metadata.Name
	if ref.Container == "" && len(pod.Status.ContainerStatuses) > 0 {
		ref.Container = pod.Status.ContainerStatuses[0].Name
	}
	containerID := pickContainerID(pod, ref.Container)
	if containerID == "" {
		return nil, fmt.Errorf("container %q not found in pod %s/%s", ref.Container, ref.Namespace, ref.Pod)
	}
	ns := strings.TrimSpace(opts.CollectorNamespace)
	if ns == "" {
		ns = "kube-system"
	}
	image := strings.TrimSpace(opts.CollectorImage)
	if image == "" {
		image = "busybox:1.36"
	}
	collectorName := collectorPodName(ref.Namespace, ref.Pod)
	if err := ensureCollectorPod(ctx, k, ns, collectorName, pod.Spec.NodeName, image); err != nil {
		return nil, err
	}
	if !opts.KeepCollector {
		defer deleteCollectorPod(context.Background(), k, ns, collectorName)
	}
	hostPID, err := findHostPIDInCollector(ctx, k, ns, collectorName, containerID)
	if err != nil {
		return nil, err
	}
	collectOne := func() (*Report, error) {
		tmp, err := snapshotRemoteProc(ctx, k, ns, collectorName, hostPID)
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tmp)
		rep, err := Collect(Options{
			PID:        hostPID,
			Namespace:  ref.Namespace,
			Pod:        ref.Pod,
			Container:  ref.Container,
			ProcRoot:   filepath.Join(tmp, "proc"),
			CgroupRoot: filepath.Join(tmp, "cgroup"),
		})
		if err != nil {
			return nil, err
		}
		rep.Target.Node = pod.Spec.NodeName
		rep.Target.ContainerID = containerID
		rep.Target.Target = ref.Namespace + "/" + ref.Pod + "/" + ref.Container
		rep.Target.Source = "kubernetes"
		rep.Summary = SummarizeReport(rep, nil)
		return rep, nil
	}
	wr, err := CollectWatchWith(ctx, interval, count, collectOne)
	if err != nil {
		return nil, err
	}
	wr.Target.Namespace = ref.Namespace
	wr.Target.Pod = ref.Pod
	wr.Target.Container = ref.Container
	wr.Target.Node = pod.Spec.NodeName
	wr.Target.ContainerID = containerID
	wr.Target.Target = ref.Namespace + "/" + ref.Pod + "/" + ref.Container
	wr.Target.Source = "kubernetes"
	wr.Summary = SummarizeWatchReport(wr)
	return wr, nil
}

func kubectlBin(bin string) string {
	if strings.TrimSpace(bin) != "" {
		return strings.TrimSpace(bin)
	}
	return "kubectl"
}

func currentKubectlNamespace(ctx context.Context, kubectl string) string {
	out, err := kubectlOutput(ctx, kubectl, 8*time.Second, "config", "view", "--minify", "-o", "jsonpath={..namespace}")
	if err != nil || strings.TrimSpace(out) == "" {
		return "default"
	}
	return strings.TrimSpace(out)
}

func loadPod(ctx context.Context, kubectl, ns, pod string) (kubectlPodJSON, error) {
	if strings.TrimSpace(ns) == "" {
		ns = "default"
	}
	out, err := kubectlOutput(ctx, kubectl, 20*time.Second, "get", "pod", "-n", ns, pod, "-o", "json")
	if err != nil {
		return kubectlPodJSON{}, err
	}
	var parsed kubectlPodJSON
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return kubectlPodJSON{}, err
	}
	return parsed, nil
}

func loadPodAcrossNamespaces(ctx context.Context, kubectl, pod string) (kubectlPodJSON, error) {
	out, err := kubectlOutput(ctx, kubectl, 25*time.Second, "get", "pods", "-A", "--field-selector=metadata.name="+pod, "-o", "json")
	if err != nil {
		return kubectlPodJSON{}, err
	}
	var parsed struct {
		Items []kubectlPodJSON `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return kubectlPodJSON{}, err
	}
	if len(parsed.Items) == 0 {
		return kubectlPodJSON{}, fmt.Errorf("pod %q not found", pod)
	}
	if len(parsed.Items) > 1 {
		return kubectlPodJSON{}, fmt.Errorf("pod %q exists in multiple namespaces, use namespace/pod", pod)
	}
	return parsed.Items[0], nil
}

func pickContainerID(pod kubectlPodJSON, name string) string {
	name = strings.TrimSpace(name)
	for _, c := range pod.Status.ContainerStatuses {
		if name == "" || c.Name == name {
			return c.ContainerID
		}
	}
	return ""
}

func collectorPodName(ns, pod string) string {
	sum := sha1.Sum([]byte(ns + "/" + pod + "/" + strconv.FormatInt(time.Now().UnixNano(), 10)))
	return "ai-sre-go-diag-" + hex.EncodeToString(sum[:])[:10]
}

func ensureCollectorPod(ctx context.Context, kubectl, ns, name, node, image string) error {
	manifest := fmt.Sprintf(`apiVersion: v1
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
	if _, err := kubectlInput(ctx, kubectl, 30*time.Second, manifest, "apply", "-f", "-"); err != nil {
		return err
	}
	_, err := kubectlOutput(ctx, kubectl, 75*time.Second, "-n", ns, "wait", "--for=condition=Ready", "pod/"+name, "--timeout=60s")
	return err
}

func deleteCollectorPod(ctx context.Context, kubectl, ns, name string) {
	_, _ = kubectlOutput(ctx, kubectl, 20*time.Second, "-n", ns, "delete", "pod", name, "--ignore-not-found=true", "--wait=false")
}

func findHostPIDInCollector(ctx context.Context, kubectl, ns, pod, containerID string) (int, error) {
	id := normalizeContainerID(containerID)
	if len(id) > 16 {
		id = id[:16]
	}
	script := fmt.Sprintf(`for f in /host/proc/[0-9]*/cgroup; do grep -q %q "$f" 2>/dev/null || continue; p="${f#/host/proc/}"; echo "${p%%/*}"; exit 0; done; exit 2`, id)
	out, err := kubectlOutput(ctx, kubectl, 30*time.Second, "-n", ns, "exec", pod, "--", "sh", "-c", script)
	if err != nil {
		return 0, fmt.Errorf("resolve host pid: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid host pid %q", strings.TrimSpace(out))
	}
	return pid, nil
}

func snapshotRemoteProc(ctx context.Context, kubectl, ns, pod string, pid int) (string, error) {
	tmp, err := os.MkdirTemp("", "ai-sre-go-runtime-*")
	if err != nil {
		return "", err
	}
	pidDir := filepath.Join(tmp, "proc", strconv.Itoa(pid))
	if err := os.MkdirAll(filepath.Join(pidDir, "fd"), 0o755); err != nil {
		_ = os.RemoveAll(tmp)
		return "", err
	}
	files := []string{"status", "smaps_rollup", "stat", "limits", "maps", "cgroup"}
	for _, name := range files {
		if err := fetchCollectorFile(ctx, kubectl, ns, pod, path.Join("/host/proc", strconv.Itoa(pid), name), filepath.Join(pidDir, name)); err != nil {
			_ = os.WriteFile(filepath.Join(pidDir, name), []byte(""), 0o644)
		}
	}
	out, _ := kubectlOutput(ctx, kubectl, 15*time.Second, "-n", ns, "exec", pod, "--", "sh", "-c", fmt.Sprintf("ls -1 /host/proc/%d/fd 2>/dev/null || true", pid))
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		_ = os.WriteFile(filepath.Join(pidDir, "fd", line), []byte{}, 0o644)
	}
	if refs, err := parseCgroups(filepath.Join(pidDir, "cgroup")); err == nil {
		fetchRemoteCgroupFiles(ctx, kubectl, ns, pod, tmp, refs)
	}
	return tmp, nil
}

func fetchRemoteCgroupFiles(ctx context.Context, kubectl, ns, pod, tmp string, refs []CgroupRef) {
	for _, ref := range refs {
		cgPath := strings.TrimPrefix(ref.Path, "/")
		if len(ref.Controllers) == 0 {
			for _, name := range []string{"memory.current", "memory.max", "memory.high", "cpu.stat"} {
				_ = fetchCollectorFile(ctx, kubectl, ns, pod, path.Join("/host/sys/fs/cgroup", cgPath, name), filepath.Join(tmp, "cgroup", cgPath, name))
			}
			continue
		}
		for _, ctrl := range ref.Controllers {
			switch ctrl {
			case "memory":
				for _, name := range []string{"memory.usage_in_bytes", "memory.limit_in_bytes"} {
					_ = fetchCollectorFile(ctx, kubectl, ns, pod, path.Join("/host/sys/fs/cgroup/memory", cgPath, name), filepath.Join(tmp, "cgroup", "memory", cgPath, name))
				}
			case "cpu", "cpuacct":
				for _, base := range []string{"cpu,cpuacct", "cpuacct"} {
					_ = fetchCollectorFile(ctx, kubectl, ns, pod, path.Join("/host/sys/fs/cgroup", base, cgPath, "cpuacct.usage"), filepath.Join(tmp, "cgroup", base, cgPath, "cpuacct.usage"))
				}
			}
		}
	}
}

func fetchCollectorFile(ctx context.Context, kubectl, ns, pod, remote, local string) error {
	out, err := kubectlOutput(ctx, kubectl, 15*time.Second, "-n", ns, "exec", pod, "--", "sh", "-c", "cat "+shellQuote(remote)+" 2>/dev/null")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
		return err
	}
	return os.WriteFile(local, []byte(out), 0o644)
}

func normalizeContainerID(id string) string {
	if _, rest, ok := strings.Cut(id, "://"); ok {
		return rest
	}
	return id
}

func kubectlOutput(ctx context.Context, kubectl string, timeout time.Duration, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, kubectl, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil {
		return out, fmt.Errorf("kubectl %s: %w: %s", strings.Join(args, " "), err, out)
	}
	return out, nil
}

func kubectlInput(ctx context.Context, kubectl string, timeout time.Duration, input string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, kubectl, args...)
	cmd.Stdin = strings.NewReader(input)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil {
		return out, fmt.Errorf("kubectl %s: %w: %s", strings.Join(args, " "), err, out)
	}
	return out, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
