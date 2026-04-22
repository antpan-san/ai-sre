package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"ft-backend/handlers"
)

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func main() {
	out := flag.String("o", "opsfleet-k8s.zip", "output zip path")
	cluster := flag.String("cluster", "test1", "cluster name")
	version := flag.String("version", "v1.28.15", "kubernetes version")
	master := flag.String("master", "", "master hosts, comma-separated (required)")
	worker := flag.String("worker", "", "worker hosts, comma-separated")
	imageSource := flag.String("imageSource", "aliyun", "image/binary mirror: default|aliyun|tencent|custom")
	arch := flag.String("arch", "arm64", "target cpu arch: amd64|arm64 (match uname -m on nodes)")
	preCleanup := flag.Bool("preCleanup", false, "embed install.sh default: run Step 0 pre_cleanup playbook (non-interactive)")
	downloadDomain := flag.String("downloadDomain", "", "override inventory download_domain (empty = use ansible-agent/inventory/group_vars/all.yml default)")
	downloadProtocol := flag.String("downloadProtocol", "", "override download_protocol, e.g. http:// or https://")
	networkPlugin := flag.String("networkPlugin", "calico", "CNI: calico|flannel (与控制台默认一致，勿留空否则合并包曾回退为 flannel)")
	pauseImage := flag.String("pauseImage", "", "optional kubelet pause image, e.g. registry.k8s.io/pause:3.10")
	flag.Parse()

	masters := splitCSV(*master)
	req := handlers.K8sDeployRequest{
		ClusterName:         *cluster,
		Version:             *version,
		ArchVersion:         *arch,
		ImageSource:         *imageSource,
		MasterHosts:         masters,
		WorkerHosts:         splitCSV(*worker),
		EnableRBAC:          true,
		DefaultStorageClass: true,
		PreDeployCleanup:    *preCleanup,
		DownloadDomain:      *downloadDomain,
		DownloadProtocol:    *downloadProtocol,
		NetworkPlugin:       *networkPlugin,
		PauseImage:          *pauseImage,
	}
	data, err := handlers.BuildK8sOfflineZip(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, data, 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes)\n", *out, len(data))
}
