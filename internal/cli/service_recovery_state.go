package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const serviceRecoveryStateDir = "/var/lib/opsfleet/service-deploy"

// ServiceRecoveryState captures service install failure context for `ops service recover`.
type ServiceRecoveryState struct {
	UpdatedAt         time.Time `json:"updated_at"`
	Service           string    `json:"service"`
	Operation         string    `json:"operation,omitempty"`
	APIURL            string    `json:"api_url,omitempty"`
	DeployID          string    `json:"deploy_id,omitempty"`
	FailedStep        string    `json:"failed_step,omitempty"`
	ExitCode          int       `json:"exit_code,omitempty"`
	LogTail           string    `json:"log_tail,omitempty"`
	LastError         string    `json:"last_error,omitempty"`
	FailedExecutionID string    `json:"failed_execution_id,omitempty"`
}

func serviceRecoveryStatePath(service string) string {
	service = strings.TrimSpace(strings.ToLower(service))
	return filepath.Join(serviceRecoveryStateDir, "recovery-"+service+".json")
}

func loadServiceRecoveryState(service string) (*ServiceRecoveryState, error) {
	b, err := os.ReadFile(serviceRecoveryStatePath(service))
	if err != nil {
		return nil, err
	}
	var st ServiceRecoveryState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func saveServiceRecoveryState(st ServiceRecoveryState) error {
	if os.Geteuid() != 0 {
		return nil
	}
	st.UpdatedAt = time.Now().UTC()
	_ = os.MkdirAll(serviceRecoveryStateDir, 0755)
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(serviceRecoveryStatePath(st.Service), raw, 0644)
}

func captureServiceInstallFailure(service, operation, failedStep string, exitCode int, lastErr string) {
	service = strings.TrimSpace(strings.ToLower(service))
	st := ServiceRecoveryState{
		Service:           service,
		Operation:         firstNonEmptyStr(operation, "install"),
		FailedStep:        strings.TrimSpace(failedStep),
		ExitCode:          exitCode,
		LastError:         strings.TrimSpace(lastErr),
		LogTail:           strings.TrimSpace(lastErr),
		FailedExecutionID: strings.TrimSpace(ActiveExecutionRecordID()),
	}
	if dep, err := loadServiceDeploymentState(service); err == nil && dep != nil {
		st.APIURL = dep.APIURL
		st.DeployID = dep.DeployID
	}
	_ = saveServiceRecoveryState(st)
}

func removeServiceRecoveryState(service string) error {
	err := os.Remove(serviceRecoveryStatePath(service))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
