package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// K8sRecoveryStatePath persists install failure context for `ops k8s recover`.
const K8sRecoveryStatePath = k8sStateDir + "/recovery-state.json"

// K8sRecoveryState captures failure context from install/bootstrap for automated recovery.
type K8sRecoveryState struct {
	UpdatedAt   time.Time `json:"updated_at"`
	Operation   string    `json:"operation,omitempty"`
	InstallRef  string    `json:"install_ref,omitempty"`
	APIBase     string    `json:"api_base,omitempty"`
	InviteID    string    `json:"invite_id,omitempty"`
	BundleRoot  string    `json:"bundle_root,omitempty"`
	LastBundle  string    `json:"last_bundle,omitempty"`
	FailedStep  string    `json:"failed_step,omitempty"`
	ExitCode    int       `json:"exit_code,omitempty"`
	LogTail     string    `json:"log_tail,omitempty"`
	Inventory   string    `json:"inventory_path,omitempty"`
	AnsibleRoot string    `json:"ansible_root,omitempty"`
}

func loadK8sRecoveryState() (*K8sRecoveryState, error) {
	b, err := os.ReadFile(K8sRecoveryStatePath)
	if err != nil {
		return nil, err
	}
	var st K8sRecoveryState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func saveK8sRecoveryState(st K8sRecoveryState) error {
	if os.Geteuid() != 0 {
		return nil
	}
	st.UpdatedAt = time.Now().UTC()
	if strings.TrimSpace(st.LastBundle) == "" {
		st.LastBundle = K8sLastBundlePath
	}
	if b, err := os.ReadFile(filepath.Join(strings.TrimSpace(st.BundleRoot), ".opsfleet-k8s-state")); err == nil {
		tail := string(b)
		if len(tail) > 8000 {
			tail = tail[len(tail)-8000:]
		}
		if strings.TrimSpace(st.LogTail) == "" {
			st.LogTail = tail
		}
	}
	_ = os.MkdirAll(k8sStateDir, 0755)
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(K8sRecoveryStatePath, raw, 0644)
}

func captureK8sInstallFailure(bundleRoot, installRef, operation string, exitCode int, failedStep string) {
	root := strings.TrimSpace(bundleRoot)
	if root == "" {
		root = K8sLastBundlePath
	}
	st := K8sRecoveryState{
		Operation:   firstNonEmptyStr(operation, "install"),
		InstallRef:  strings.TrimSpace(installRef),
		BundleRoot:  root,
		LastBundle:  K8sLastBundlePath,
		FailedStep:  strings.TrimSpace(failedStep),
		ExitCode:    exitCode,
		Inventory:   filepath.Join(root, "inventory", "hosts.ini"),
		AnsibleRoot: filepath.Join(root, "ansible-agent"),
	}
	if st.InstallRef != "" {
		if wire, err := decodeInstallRefV1(st.InstallRef); err == nil {
			st.APIBase = wire.B
			st.InviteID = wire.I
		}
	}
	_ = saveK8sRecoveryState(st)
}

func mergeRecoveryBundleRoot(st *K8sRecoveryState) string {
	var roots []string
	if st != nil {
		roots = append(roots, st.BundleRoot, st.LastBundle)
	}
	roots = append(roots, K8sLastBundlePath)
	for _, p := range roots {
		p = strings.TrimSpace(p)
		if p != "" && isLocalK8sBundleRoot(p) {
			return p
		}
	}
	return ""
}
