// opsfleet-k8s-mirror-serve：K8s 内网制品站「边下边存」
//
// 与 deploy/k8s-mirror/k8s-mirror-sync.sh 的目录布局、上游 URL 规则一致；供 Nginx
// 在 try_files 未命中时反代，将缺失的 kubernetes/etcd/cni 制品从公网拉入 MIRROR_ROOT
// 并永久缓存，随后本地读盘即与全量预同步一致。
package main

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

var (
	mirrorRoot   string
	k8sUpstream  string
	etcdUpstream string
	cniUpstream  string
	listen       string
	sf           singleflight.Group
)

func init() {
	mirrorRoot = envOr("MIRROR_ROOT", "/var/lib/opsfleet-k8s-mirror")
	if a, err := filepath.Abs(mirrorRoot); err == nil {
		mirrorRoot = a
	}
	k8sUpstream = strings.TrimRight(envOr("K8S_UPSTREAM", "https://dl.k8s.io"), "/")
	etcdUpstream = strings.TrimRight(envOr("ETCD_UPSTREAM", "https://github.com/etcd-io/etcd/releases/download"), "/")
	cniUpstream = strings.TrimRight(envOr("CNI_UPSTREAM", "https://github.com/containernetworking/plugins/releases/download"), "/")
	listen = envOr("LISTEN", "127.0.0.1:8090")
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func isAllowed(p string) bool {
	for _, pre := range []string{"/kubernetes/", "/etcd/", "/cni-plugins/"} {
		if strings.HasPrefix(p, pre) {
			return !strings.Contains(p, "..")
		}
	}
	return false
}

func underMirror(mirror, abs string) bool {
	m, err := filepath.EvalSymlinks(mirror)
	if err != nil {
		m = mirror
	}
	a, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(m, a)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// resolveUpstream 将 HTTP 请求路径（与 MIRROR_ROOT 下相对布局一致）映射为 k8s-mirror-sync.sh 中使用的公网 URL。
func resolveUpstream(urlPath string) (string, error) {
	p := path.Clean(urlPath)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("path too short")
	}
	switch parts[1] {
	case "kubernetes":
		if len(parts) != 5 {
			return "", fmt.Errorf("invalid kubernetes path")
		}
		ver, arch, name := parts[2], parts[3], parts[4]
		if !strings.HasPrefix(ver, "v") {
			return "", fmt.Errorf("invalid version")
		}
		if !strings.HasPrefix(name, "kubernetes-server-linux-") || !strings.HasSuffix(name, ".tar.gz") {
			return "", fmt.Errorf("invalid package name")
		}
		if !strings.HasSuffix(name, "linux-"+arch+".tar.gz") {
			return "", fmt.Errorf("arch mismatch in filename")
		}
		return fmt.Sprintf("%s/%s/%s", k8sUpstream, ver, name), nil
	case "etcd":
		if len(parts) != 4 {
			return "", fmt.Errorf("invalid etcd path")
		}
		ver, name := parts[2], parts[3]
		if !strings.HasPrefix(ver, "v") {
			return "", fmt.Errorf("invalid etcd version")
		}
		if !strings.HasPrefix(name, "etcd-"+ver+"-linux-") || !strings.HasSuffix(name, ".tar.gz") {
			return "", fmt.Errorf("invalid etcd package name")
		}
		return fmt.Sprintf("%s/%s/%s", etcdUpstream, ver, name), nil
	case "cni-plugins":
		if len(parts) != 4 {
			return "", fmt.Errorf("invalid cni path")
		}
		ver, name := parts[2], parts[3]
		if !strings.HasPrefix(ver, "v") {
			return "", fmt.Errorf("invalid cni version")
		}
		if !strings.HasPrefix(name, "cni-plugins-linux-") || !strings.HasSuffix(name, ".tgz") {
			return "", fmt.Errorf("invalid cni package name")
		}
		if !strings.Contains(name, ver) {
			return "", fmt.Errorf("cni version in path/filename mismatch")
		}
		return fmt.Sprintf("%s/%s/%s", cniUpstream, ver, name), nil
	default:
		return "", fmt.Errorf("unknown prefix")
	}
}

// k8s 下载后拉取同目录 .sha512（与 sync 一致），有有效 128 位十六进制则强校验，否则仅落盘/跳过。
func finalizeK8sArtifact(ctx context.Context, filePath, k8sURL string, client *http.Client) error {
	shaURL := k8sURL + ".sha512"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shaURL, nil)
	if err != nil {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return nil
	}
	_ = os.WriteFile(filePath+".sha512", b, 0644)
	line := strings.TrimSpace(string(b))
	fields := strings.Fields(line)
	if len(fields) < 1 {
		return nil
	}
	wantHex := strings.ToLower(fields[0])
	if len(wantHex) != 128 {
		return nil
	}
	f, err := os.Open(filePath)
	if err != nil {
		_ = os.Remove(filePath)
		_ = os.Remove(filePath + ".sha512")
		return err
	}
	h := sha512.New()
	_, err = io.Copy(h, f)
	_ = f.Close()
	if err != nil {
		_ = os.Remove(filePath)
		_ = os.Remove(filePath + ".sha512")
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != wantHex {
		_ = os.Remove(filePath)
		_ = os.Remove(filePath + ".sha512")
		return fmt.Errorf("sha512 mismatch vs upstream .sha512")
	}
	return nil
}

func downloadToFile(ctx context.Context, url, dest string, client *http.Client) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	tmp := dest + ".downloading"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_ = f.Close()
		_ = os.Remove(tmp)
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func ensureArtifact(ctx context.Context, urlPath, local string, client *http.Client) error {
	up, err := resolveUpstream(urlPath)
	if err != nil {
		return err
	}
	if err := downloadToFile(ctx, up, local, client); err != nil {
		return err
	}
	if strings.HasPrefix(path.Clean(urlPath), "/kubernetes/") {
		if err := finalizeK8sArtifact(ctx, local, up, client); err != nil {
			return err
		}
	}
	return nil
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p := path.Clean(r.URL.Path)
	if p == "" || p == "/" {
		http.NotFound(w, r)
		return
	}
	if !isAllowed(p) {
		http.NotFound(w, r)
		return
	}
	local := filepath.Join(mirrorRoot, strings.TrimLeft(p, "/"))
	if st, err := os.Stat(local); err == nil && !st.IsDir() {
		http.ServeFile(w, r, local)
		return
	}
	abs, err := filepath.Abs(local)
	if err != nil || !underMirror(mirrorRoot, abs) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	client := &http.Client{Timeout: 0}
	ctx := r.Context()
	_, err, _ = sf.Do(p, func() (interface{}, error) {
		if st, e := os.Stat(local); e == nil && !st.IsDir() {
			return nil, nil
		}
		log.Printf("[mirror-serve] on-demand fetch: %s", p)
		if e := ensureArtifact(ctx, p, local, client); e != nil {
			log.Printf("[mirror-serve] fetch failed: %s: %v", p, e)
			return nil, e
		}
		log.Printf("[mirror-serve] cached: %s", local)
		return nil, nil
	})
	if err != nil {
		http.Error(w, "upstream: "+err.Error(), http.StatusBadGateway)
		return
	}
	if _, e := os.Stat(local); e != nil {
		http.Error(w, "file missing after fetch", http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, local)
}

func main() {
	_ = os.MkdirAll(mirrorRoot, 0755)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/kubernetes/", handler)
	mux.HandleFunc("/etcd/", handler)
	mux.HandleFunc("/cni-plugins/", handler)
	srv := &http.Server{Addr: listen, Handler: mux, ReadHeaderTimeout: 30 * time.Second}
	log.Printf("opsfleet-k8s-mirror-serve: listen=%s MIRROR_ROOT=%s", listen, mirrorRoot)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
