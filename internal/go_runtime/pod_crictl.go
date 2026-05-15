package go_runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CrictlPodPIDResolver resolves a Kubernetes pod container to the host PID using crictl
// (must run on the node where the pod is scheduled).
type CrictlPodPIDResolver struct {
	// Bin is the crictl executable path; empty means "crictl" from PATH.
	Bin string
}

func (r CrictlPodPIDResolver) crictl() string {
	if strings.TrimSpace(r.Bin) != "" {
		return strings.TrimSpace(r.Bin)
	}
	return "crictl"
}

// DefaultPodPIDResolver returns the production resolver (crictl-based).
func DefaultPodPIDResolver() PodPIDResolver {
	return CrictlPodPIDResolver{}
}

type crictlPodsJSON struct {
	Items []struct {
		ID       string `json:"id"`
		Metadata struct {
			Namespace string            `json:"namespace"`
			Name      string            `json:"name"`
			Labels    map[string]string `json:"labels"`
		} `json:"metadata"`
	} `json:"items"`
}

type crictlContainersJSON struct {
	Containers []struct {
		ID       string `json:"id"`
		Metadata struct {
			Name   string            `json:"name"`
			Labels map[string]string `json:"labels"`
		} `json:"metadata"`
	} `json:"containers"`
}

type crictlInspect struct {
	Info struct {
		Pid int `json:"pid"`
	} `json:"info"`
}

func (r CrictlPodPIDResolver) Resolve(namespace, pod, container string) (int, error) {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "default"
	}
	pod = strings.TrimSpace(pod)
	if pod == "" {
		return 0, fmt.Errorf("pod name is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	sandboxID, err := r.findPodSandboxID(ctx, ns, pod)
	if err != nil {
		return 0, err
	}
	containerID, err := r.pickContainerID(ctx, sandboxID, container)
	if err != nil {
		return 0, err
	}
	pid, err := r.inspectPID(ctx, containerID)
	if err != nil {
		return 0, err
	}
	if pid <= 0 {
		return 0, fmt.Errorf("crictl inspect returned invalid pid=%d for container %s", pid, containerID)
	}
	return pid, nil
}

func (r CrictlPodPIDResolver) findPodSandboxID(ctx context.Context, namespace, pod string) (string, error) {
	cmd := exec.CommandContext(ctx, r.crictl(), "pods", "--namespace", namespace, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("crictl pods: %w", err)
	}
	var parsed crictlPodsJSON
	if err := json.Unmarshal(out, &parsed); err != nil {
		return "", fmt.Errorf("crictl pods json: %w", err)
	}
	const labelPod = "io.kubernetes.pod.name"
	const labelNS = "io.kubernetes.pod.namespace"
	for _, it := range parsed.Items {
		labels := it.Metadata.Labels
		nameMatch := labels[labelPod] == pod || strings.Contains(it.Metadata.Name, pod)
		nsMatch := labels[labelNS] == "" || labels[labelNS] == namespace
		if nameMatch && nsMatch {
			if strings.TrimSpace(it.ID) != "" {
				return it.ID, nil
			}
		}
	}
	return "", fmt.Errorf("no crictl pod sandbox found for namespace=%q pod=%q", namespace, pod)
}

func (r CrictlPodPIDResolver) pickContainerID(ctx context.Context, sandboxID, container string) (string, error) {
	cmd := exec.CommandContext(ctx, r.crictl(), "ps", "-a", "--pod", sandboxID, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("crictl ps: %w", err)
	}
	var parsed crictlContainersJSON
	if err := json.Unmarshal(out, &parsed); err != nil {
		return "", fmt.Errorf("crictl ps json: %w", err)
	}
	container = strings.TrimSpace(container)
	if len(parsed.Containers) == 0 {
		return "", fmt.Errorf("no containers in pod sandbox %s", sandboxID)
	}
	if container == "" {
		return parsed.Containers[0].ID, nil
	}
	const labelCtr = "io.kubernetes.container.name"
	for _, c := range parsed.Containers {
		if c.Metadata.Name == container || c.Metadata.Labels[labelCtr] == container {
			if strings.TrimSpace(c.ID) != "" {
				return c.ID, nil
			}
		}
	}
	return "", fmt.Errorf("container %q not found in pod sandbox", container)
}

func (r CrictlPodPIDResolver) inspectPID(ctx context.Context, containerID string) (int, error) {
	cmd := exec.CommandContext(ctx, r.crictl(), "inspect", containerID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("crictl inspect: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var parsed crictlInspect
	if err := json.Unmarshal(out, &parsed); err != nil {
		return 0, fmt.Errorf("crictl inspect json: %w", err)
	}
	return parsed.Info.Pid, nil
}
