package cli

import (
	"fmt"
	"os"
	"strings"

	goruntime "github.com/panshuai/ai-sre/internal/go_runtime"
	"github.com/spf13/cobra"
)

type goRuntimeCLIOptions struct {
	PID        int
	Namespace  string
	Pod        string
	Container  string
	ProcRoot   string
	CgroupRoot string
	JSON       bool
}

func diagnoseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "非 AI 本地快诊工具",
	}
	cmd.AddCommand(goProcessDiagnoseCmd())
	return cmd
}

func goProcessDiagnoseCmd() *cobra.Command {
	var opts goRuntimeCLIOptions
	cmd := &cobra.Command{
		Use:   "go-process",
		Short: "非侵入式 Go 进程运行时诊断",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGoRuntimeCLI(opts)
		},
	}
	bindGoRuntimeFlags(cmd, &opts)
	cmd.Example = fmt.Sprintf(`  %s diagnose go-process --pid 1234
  %s diagnose go-process --pid 1234 -o json
  %s diagnose go-process --namespace default --pod api-0 --container app`, progName, progName, progName)
	return cmd
}

func runGoRuntimeAnalyze(topic string, opts goRuntimeCLIOptions) error {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "go-runtime", "pod-go":
		return runGoRuntimeCLI(opts)
	default:
		return fmt.Errorf("unsupported go runtime topic %q", topic)
	}
}

func runGoRuntimeCLI(opts goRuntimeCLIOptions) error {
	if opts.PID <= 0 {
		if strings.TrimSpace(opts.Pod) != "" {
			_, err := goruntime.UnimplementedPodPIDResolver{}.Resolve(opts.Namespace, opts.Pod, opts.Container)
			return err
		}
		return fmt.Errorf("请提供 --pid；Pod 定位接口已预留但 MVP 暂未实现")
	}
	report, err := goruntime.Collect(goruntime.Options{
		PID:        opts.PID,
		Namespace:  opts.Namespace,
		Pod:        opts.Pod,
		Container:  opts.Container,
		ProcRoot:   opts.ProcRoot,
		CgroupRoot: opts.CgroupRoot,
	})
	if err != nil {
		return err
	}
	if opts.JSON || strings.EqualFold(outputFormat, "json") {
		return goruntime.WriteJSON(os.Stdout, report)
	}
	return goruntime.WriteText(os.Stdout, report)
}

func bindGoRuntimeFlags(cmd *cobra.Command, opts *goRuntimeCLIOptions) {
	cmd.Flags().IntVar(&opts.PID, "pid", 0, "本地宿主机进程 PID")
	cmd.Flags().StringVarP(&opts.Namespace, "namespace", "n", "", "Kubernetes namespace（Pod 定位预留）")
	cmd.Flags().StringVar(&opts.Pod, "pod", "", "Kubernetes Pod 名称（MVP 预留）")
	cmd.Flags().StringVar(&opts.Container, "container", "", "Kubernetes container 名称（MVP 预留）")
	cmd.Flags().StringVar(&opts.ProcRoot, "proc-root", "/proc", "procfs 根目录")
	cmd.Flags().StringVar(&opts.CgroupRoot, "cgroup-root", "/sys/fs/cgroup", "cgroupfs 根目录")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "输出机器可读 JSON")
}

func bindGoRuntimeAnalyzeFlags(cmd *cobra.Command, opts *goRuntimeCLIOptions) {
	cmd.Flags().IntVar(&opts.PID, "pid", 0, "Go runtime: 本地宿主机进程 PID")
	cmd.Flags().StringVar(&opts.Container, "container", "", "Go runtime: Kubernetes container 名称（MVP 预留）")
	cmd.Flags().StringVar(&opts.ProcRoot, "proc-root", "/proc", "Go runtime: procfs 根目录")
	cmd.Flags().StringVar(&opts.CgroupRoot, "cgroup-root", "/sys/fs/cgroup", "Go runtime: cgroupfs 根目录")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Go runtime: 输出机器可读 JSON")
}
