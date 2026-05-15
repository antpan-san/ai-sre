package go_runtime

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const maxProbeBytes = 85_000

// GatherPodProbeBundle runs read-only kubectl against the target pod (describe/events/logs).
func GatherPodProbeBundle(ctx context.Context, kubectl, namespace, pod string) map[string]string {
	namespace = strings.TrimSpace(namespace)
	pod = strings.TrimSpace(pod)
	if namespace == "" || pod == "" {
		return nil
	}
	kubectl = kubectlBin(kubectl)
	out := map[string]string{}
	put := func(key, body string) {
		body = truncateProbe(body, maxProbeBytes)
		if strings.TrimSpace(body) != "" {
			out[key] = body
		}
	}
	put("kubectl_focus_pod_ref", namespace+"/"+pod)
	put("kubectl_focus_describe", kubectlOutputSafe(ctx, kubectl, 35*time.Second, "describe", "pod", "-n", namespace, pod))
	ev := kubectlOutputSafe(ctx, kubectl, 18*time.Second, "get", "events", "-n", namespace, "--field-selector=involvedObject.name="+pod, "-o", "wide")
	if strings.Contains(strings.ToLower(ev), "unknown field") || strings.Contains(strings.ToLower(ev), "invalid field") {
		ev = kubectlOutputSafe(ctx, kubectl, 20*time.Second, "get", "events", "-n", namespace, "-o", "wide", "--sort-by=.metadata.creationTimestamp")
		if len(ev) > 18_000 {
			ev = "... [tail]\n" + ev[len(ev)-18_000:]
		}
	}
	put("kubectl_focus_events", ev)
	put("kubectl_focus_logs_current", kubectlOutputSafe(ctx, kubectl, 35*time.Second,
		"logs", "-n", namespace, pod, "--all-containers=true", "--tail=600"))
	put("kubectl_focus_logs_previous", kubectlOutputSafe(ctx, kubectl, 28*time.Second,
		"logs", "-n", namespace, pod, "--all-containers=true", "--previous", "--tail=400"))
	return out
}

// GatherCollectorProbeBundle captures collector pod failure context.
func GatherCollectorProbeBundle(ctx context.Context, kubectl, namespace, pod string) map[string]string {
	namespace = strings.TrimSpace(namespace)
	pod = strings.TrimSpace(pod)
	if namespace == "" || pod == "" {
		return nil
	}
	kubectl = kubectlBin(kubectl)
	return map[string]string{
		"kubectl_collector_describe": kubectlOutputSafe(ctx, kubectl, 20*time.Second, "describe", "pod", "-n", namespace, pod),
	}
}

func kubectlOutputSafe(ctx context.Context, kubectl string, timeout time.Duration, args ...string) string {
	out, err := kubectlOutput(ctx, kubectl, timeout, args...)
	if err != nil {
		if strings.TrimSpace(out) != "" {
			return out
		}
		return err.Error()
	}
	return out
}

func truncateProbe(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "\n... [truncated]\n"
}

func mergeProbeBundles(parts ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, p := range parts {
		for k, v := range p {
			if strings.TrimSpace(v) == "" {
				continue
			}
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func attachProbeBundle(wr *WatchReport, bundle map[string]string) {
	if wr == nil || len(bundle) == 0 {
		return
	}
	if wr.ProbeBundle == nil {
		wr.ProbeBundle = map[string]string{}
	}
	for k, v := range bundle {
		wr.ProbeBundle[k] = v
	}
}

func probeBundleText(bundle map[string]string, max int) string {
	if len(bundle) == 0 {
		return ""
	}
	var b strings.Builder
	for _, key := range []string{
		"kubectl_focus_pod_ref",
		"kubectl_focus_describe",
		"kubectl_focus_events",
		"kubectl_focus_logs_current",
		"kubectl_focus_logs_previous",
		"kubectl_collector_describe",
	} {
		if v := strings.TrimSpace(bundle[key]); v != "" {
			b.WriteString("### ")
			b.WriteString(key)
			b.WriteString("\n")
			b.WriteString(v)
			b.WriteString("\n\n")
		}
	}
	for k, v := range bundle {
		if strings.HasPrefix(k, "kubectl_focus_") || k == "kubectl_collector_describe" {
			continue
		}
		b.WriteString("### ")
		b.WriteString(k)
		b.WriteString("\n")
		b.WriteString(v)
		b.WriteString("\n\n")
	}
	return truncateProbe(b.String(), max)
}

func collectorImagePullInBundle(bundle map[string]string) bool {
	text := strings.ToLower(probeBundleText(bundle, 32_000))
	return strings.Contains(text, "imagepullbackoff") ||
		strings.Contains(text, "errimagepull") ||
		strings.Contains(text, "failed to pull image")
}

func formatProbeRef(ns, pod string) string {
	return fmt.Sprintf("%s/%s", strings.TrimSpace(ns), strings.TrimSpace(pod))
}
