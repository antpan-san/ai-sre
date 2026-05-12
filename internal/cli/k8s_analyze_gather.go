package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// maxBytesK8sEvidence caps total kubectl capture size sent to the diagnose API.
const maxBytesK8sEvidence = 130_000

type evidenceCollector struct {
	out    map[string]string
	budget int
}

func newEvidenceCollector() *evidenceCollector {
	return &evidenceCollector{out: make(map[string]string), budget: maxBytesK8sEvidence}
}

func (c *evidenceCollector) put(key, body string) {
	if c == nil || c.budget <= 0 {
		return
	}
	maxChunk := minInt(c.budget, 85_000)
	if maxChunk <= 0 {
		return
	}
	chunk := truncateBytes(body, maxChunk)
	if len(chunk) == 0 {
		return
	}
	c.out[key] = chunk
	c.budget -= len(chunk)
}

func truncateBytes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "\n... [truncated by ai-sre]\n"
}

func kubectlCombined(ctx context.Context, timeout time.Duration, args ...string) string {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, "kubectl", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil {
		if out == "" {
			return fmt.Sprintf("[kubectl %s failed: %v]", strings.Join(args, " "), err)
		}
		return out + fmt.Sprintf("\n[exit error: %v]", err)
	}
	return out
}

type podListJSON struct {
	Items []struct {
		Metadata struct {
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
		} `json:"metadata"`
	} `json:"items"`
}

// k8sAnalyzePodFlagIsIssueKeyword returns true when --pod names a scenario, not a Pod object.
func k8sAnalyzePodFlagIsIssueKeyword(pod string) bool {
	switch strings.ToLower(strings.TrimSpace(pod)) {
	case "", "pending", "crashloop", "crashloopbackoff", "instability":
		return true
	default:
		return false
	}
}

func uniqueSortedStrings(in []string) []string {
	seen := map[string]struct{}{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		seen[s] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// resolvePodAcrossNamespaces finds namespace for an exact Pod metadata.name.
func resolvePodAcrossNamespaces(ctx context.Context, podName, hintNS string) (ns string, ambiguousNote string, ok bool) {
	podName = strings.TrimSpace(podName)
	if podName == "" {
		return "", "", false
	}
	if hintNS != "" {
		name := strings.TrimSpace(kubectlCombined(ctx, 8*time.Second, "get", "pod", "-n", hintNS, podName, "-o", "jsonpath={.metadata.name}"))
		if name == podName {
			return hintNS, "", true
		}
	}
	raw := kubectlCombined(ctx, 20*time.Second, "get", "pods", "-A", "--field-selector=metadata.name="+podName, "-o", "json")
	var pl podListJSON
	if json.Unmarshal([]byte(raw), &pl) != nil || len(pl.Items) == 0 {
		return "", "", false
	}
	nss := make([]string, 0, len(pl.Items))
	for _, it := range pl.Items {
		nss = append(nss, strings.TrimSpace(it.Metadata.Namespace))
	}
	ns0 := strings.TrimSpace(pl.Items[0].Metadata.Namespace)
	if len(pl.Items) > 1 {
		return ns0, "multiple namespaces contain this pod name (using first match for deep gather): " + strings.Join(uniqueSortedStrings(nss), ", "), true
	}
	return ns0, "", true
}

func takeLastBytes(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return "... [tail]\n" + s[len(s)-n:]
}

func appendFocusedPodEvidence(ctx context.Context, col *evidenceCollector, flags map[string]string, podName string) {
	ns, amb, ok := resolvePodAcrossNamespaces(ctx, podName, strings.TrimSpace(flags["namespace"]))
	if !ok {
		col.put("kubectl_focus_resolve_error", fmt.Sprintf("未在集群中找到名为 %q 的 Pod；若是静态 Pod / 系统组件，请尝试: --namespace kube-system\n（仍会继续采集集群全景）", podName))
		return
	}
	if amb != "" {
		col.put("kubectl_focus_namespace_note", amb)
	}
	col.put("kubectl_focus_pod_ref", fmt.Sprintf("%s/%s", ns, podName))

	desc := kubectlCombined(ctx, 35*time.Second, "describe", "pod", "-n", ns, podName)
	col.put("kubectl_focus_describe", desc)

	ev := kubectlCombined(ctx, 18*time.Second, "get", "events", "-n", ns, "--field-selector=involvedObject.name="+podName, "-o", "wide")
	low := strings.ToLower(ev)
	if strings.Contains(low, "unknown field") || strings.Contains(low, "invalid field") || strings.Contains(low, "badrequest") {
		ev = kubectlCombined(ctx, 20*time.Second, "get", "events", "-n", ns, "-o", "wide", "--sort-by=.metadata.creationTimestamp")
		ev = takeLastBytes(ev, 18_000)
	}
	col.put("kubectl_focus_events", ev)

	logsCur := kubectlCombined(ctx, 35*time.Second, "logs", "-n", ns, podName, "--all-containers=true", "--tail=600")
	col.put("kubectl_focus_logs_current", logsCur)

	logsPrev := kubectlCombined(ctx, 28*time.Second, "logs", "-n", ns, podName, "--all-containers=true", "--previous", "--tail=400")
	col.put("kubectl_focus_logs_previous", logsPrev)
}

// gatherK8sDiagnoseEvidence runs read-only kubectl locally and returns map keys
// prefixed with kubectl_ for merging into analyze context. Best-effort: empty
// on missing kubectl or errors (caller still runs diagnose with flags-only).
func gatherK8sDiagnoseEvidence(ctx context.Context, flags map[string]string) map[string]string {
	col := newEvidenceCollector()
	if _, err := exec.LookPath("kubectl"); err != nil {
		return col.out
	}

	podFlag := strings.TrimSpace(flags["pod"])
	if podFlag != "" && !k8sAnalyzePodFlagIsIssueKeyword(podFlag) {
		appendFocusedPodEvidence(ctx, col, flags, podFlag)
	}

	col.put("kubectl_version", kubectlCombined(ctx, 12*time.Second, "version", "--client=true", "-o", "yaml"))
	col.put("kubectl_config_context", kubectlCombined(ctx, 8*time.Second, "config", "current-context"))
	col.put("kubectl_nodes", kubectlCombined(ctx, 15*time.Second, "get", "nodes", "-o", "wide"))
	col.put("kubectl_pods_all", kubectlCombined(ctx, 20*time.Second, "get", "pods", "-A", "-o", "wide"))

	raw := kubectlCombined(ctx, 20*time.Second, "get", "pods", "-A", "--field-selector=status.phase=Pending", "-o", "json")
	col.put("kubectl_pending_json", raw)
	var pl podListJSON
	var refs []struct {
		ns, name string
	}
	if json.Unmarshal([]byte(raw), &pl) == nil {
		for _, it := range pl.Items {
			ns := strings.TrimSpace(it.Metadata.Namespace)
			n := strings.TrimSpace(it.Metadata.Name)
			if ns != "" && n != "" {
				refs = append(refs, struct{ ns, name string }{ns, n})
			}
		}
	}
	maxDesc := 6
	if len(refs) > maxDesc {
		refs = refs[:maxDesc]
	}
	var descBuf strings.Builder
	for _, r := range refs {
		if col.budget <= 2000 {
			break
		}
		block := kubectlCombined(ctx, 25*time.Second, "describe", "pod", "-n", r.ns, r.name)
		descBuf.WriteString(fmt.Sprintf("### describe pod %s/%s\n", r.ns, r.name))
		descBuf.WriteString(block)
		descBuf.WriteString("\n\n")
	}
	if descBuf.Len() > 0 {
		col.put("kubectl_pending_describe", descBuf.String())
	}

	col.put("kubectl_events_recent", kubectlCombined(ctx, 20*time.Second, "get", "events", "-A", "--sort-by=.metadata.creationTimestamp"))

	ns := strings.TrimSpace(flags["namespace"])
	if ns != "" {
		col.put("kubectl_pods_namespace", kubectlCombined(ctx, 15*time.Second, "get", "pods", "-n", ns, "-o", "wide"))
	}

	return col.out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func hasKubectlEvidence(kv map[string]string) bool {
	if kv == nil {
		return false
	}
	for k := range kv {
		if strings.HasPrefix(k, "kubectl_") {
			return true
		}
	}
	return false
}
