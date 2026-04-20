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
