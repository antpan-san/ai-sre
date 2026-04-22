package cli

import (
	"os"
	"path/filepath"
	"strings"
)

// 控制机可记录 ofpk8s1 安装引用；卸载优先使用本机 last-bundle 副本，无需邀请仍有效。

const k8sStateDir = "/var/lib/opsfleet-k8s"
const k8sInstallRefFile = "install-ref"

// K8sDefaultUninstallWorkdir 为文档/常见安装方式使用的离线条解压根路径；
// 未使用 last-bundle 时 uninstall 会尝试该目录与 OPSFLEET_K8S_WORKDIR。
const K8sDefaultUninstallWorkdir = "/opt/opsfleet-k8s"

// K8sLastBundlePath 为 install.sh / 引导脚本在预检后同步的完整离线条根路径，
// 与平台版本、ofpk8s1 邀请是否过期无关，供 ai-sre uninstall k8s 无 id 清理。
const K8sLastBundlePath = k8sStateDir + "/last-bundle"

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
