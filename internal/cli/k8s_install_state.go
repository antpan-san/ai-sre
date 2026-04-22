package cli

import (
	"os"
	"path/filepath"
	"strings"
)

// 控制机记录 ofpk8s1 安装引用，供「ai-sre uninstall k8s」无参自适配备份。
// 与 k8s cleanup 使用同一套清理逻辑；引用须在有效期内（与 OpsFleet bundle-invite 一致）。

const k8sStateDir = "/var/lib/opsfleet-k8s"
const k8sInstallRefFile = "install-ref"

func k8sInstallRefSystemPath() string {
	return filepath.Join(k8sStateDir, k8sInstallRefFile)
}

func k8sInstallRefUserPath() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	return filepath.Join(home, ".config", "ai-sre", k8sInstallRefFile), true
}

// saveK8sInstallRef 在成功/失败安装前即写入，便于之后 uninstall k8s 能发现引用。
// root 时写入 /var/lib/opsfleet-k8s；同时写入 $HOME/.config/ai-sre/（当 HOME 可写时）。
func saveK8sInstallRef(ref string) {
	ref = strings.TrimSpace(ref)
	if ref == "" || !strings.HasPrefix(ref, installRefPrefixV1) {
		return
	}
	if os.Geteuid() == 0 {
		_ = os.MkdirAll(k8sStateDir, 0755)
		_ = os.WriteFile(k8sInstallRefSystemPath(), []byte(ref+"\n"), 0644)
	}
	if p, ok := k8sInstallRefUserPath(); ok {
		_ = os.MkdirAll(filepath.Dir(p), 0755)
		_ = os.WriteFile(p, []byte(ref+"\n"), 0600)
	}
}

// loadK8sInstallRef 按优先级：环境变量、系统状态文件、用户状态文件；返回 trim 后的一行或空。
func loadK8sInstallRef() string {
	if s := strings.TrimSpace(os.Getenv("OPSFLEET_K8S_INSTALL_REF")); s != "" {
		return s
	}
	if b, err := os.ReadFile(k8sInstallRefSystemPath()); err == nil {
		if s := strings.TrimSpace(string(b)); s != "" {
			return s
		}
	}
	if p, ok := k8sInstallRefUserPath(); ok {
		if b, err := os.ReadFile(p); err == nil {
			if s := strings.TrimSpace(string(b)); s != "" {
				return s
			}
		}
	}
	return ""
}
