package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// maxBytesK8sEvidence caps total kubectl capture size sent to the diagnose API.
const maxBytesK8sEvidence = 110_000

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

// gatherK8sDiagnoseEvidence runs read-only kubectl locally and returns map keys
// prefixed with kubectl_ for merging into analyze context. Best-effort: empty
// on missing kubectl or errors (caller still runs diagnose with flags-only).
func gatherK8sDiagnoseEvidence(ctx context.Context, flags map[string]string) map[string]string {
	out := map[string]string{}
	if _, err := exec.LookPath("kubectl"); err != nil {
		return out
	}

	budget := maxBytesK8sEvidence
	put := func(key, body string) {
		if budget <= 0 {
			return
		}
		chunk := truncateBytes(body, minInt(budget, 80_000))
		out[key] = chunk
		budget -= len(chunk)
	}

	put("kubectl_version", kubectlCombined(ctx, 12*time.Second, "version", "--client=true", "-o", "yaml"))
	put("kubectl_config_context", kubectlCombined(ctx, 8*time.Second, "config", "current-context"))
	put("kubectl_nodes", kubectlCombined(ctx, 15*time.Second, "get", "nodes", "-o", "wide"))
	put("kubectl_pods_all", kubectlCombined(ctx, 20*time.Second, "get", "pods", "-A", "-o", "wide"))

	raw := kubectlCombined(ctx, 20*time.Second, "get", "pods", "-A", "--field-selector=status.phase=Pending", "-o", "json")
	put("kubectl_pending_json", raw)
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
		if budget <= 2000 {
			break
		}
		block := kubectlCombined(ctx, 25*time.Second, "describe", "pod", "-n", r.ns, r.name)
		descBuf.WriteString(fmt.Sprintf("### describe pod %s/%s\n", r.ns, r.name))
		descBuf.WriteString(block)
		descBuf.WriteString("\n\n")
	}
	if descBuf.Len() > 0 {
		put("kubectl_pending_describe", descBuf.String())
	}

	put("kubectl_events_recent", kubectlCombined(ctx, 20*time.Second, "get", "events", "-A", "--sort-by=.metadata.creationTimestamp"))

	// Optional: namespace-scoped snapshot if user passed -n
	ns := strings.TrimSpace(flags["namespace"])
	if ns != "" {
		put("kubectl_pods_namespace", kubectlCombined(ctx, 15*time.Second, "get", "pods", "-n", ns, "-o", "wide"))
	}

	return out
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
