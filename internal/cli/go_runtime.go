package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

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

	WatchInterval time.Duration
	WatchSamples  int
	UploadURL     string
	SessionID     string
	SampleToken   string
	CrictlPath    string
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
			return runGoRuntimeCLI(cmd.Context(), opts)
		},
	}
	bindGoRuntimeFlags(cmd, &opts)
	cmd.Example = fmt.Sprintf(`  %s diagnose go-process --pid 1234
  %s diagnose go-process --pid 1234 -o json
  %s diagnose go-process -n default --pod api-0 --container app
  %s diagnose go-process --pid 1 --watch-samples 5 --watch-interval 10s -o json`, progName, progName, progName, progName)
	return cmd
}

func runGoRuntimeAnalyze(ctx context.Context, topic string, opts goRuntimeCLIOptions) error {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "go-runtime", "pod-go":
		return runGoRuntimeCLI(ctx, opts)
	default:
		return fmt.Errorf("unsupported go runtime topic %q", topic)
	}
}

func runGoRuntimeCLI(ctx context.Context, opts goRuntimeCLIOptions) error {
	pid := opts.PID
	if pid <= 0 && strings.TrimSpace(opts.Pod) != "" {
		resolver := goruntime.CrictlPodPIDResolver{Bin: opts.CrictlPath}
		var err error
		pid, err = resolver.Resolve(opts.Namespace, opts.Pod, opts.Container)
		if err != nil {
			return err
		}
	}
	if pid <= 0 {
		return fmt.Errorf("请提供 --pid，或使用 --namespace/--pod（及可选 --container）在节点上通过 crictl 解析宿主机 PID")
	}

	base := goruntime.Options{
		PID:        pid,
		Namespace:  opts.Namespace,
		Pod:        opts.Pod,
		Container:  opts.Container,
		ProcRoot:   opts.ProcRoot,
		CgroupRoot: opts.CgroupRoot,
	}

	samples := opts.WatchSamples
	if samples < 1 {
		samples = 1
	}
	interval := opts.WatchInterval
	if samples > 1 && interval <= 0 {
		interval = 10 * time.Second
	}

	useWatch := samples > 1
	var watchRep *goruntime.WatchReport
	var singleRep *goruntime.Report
	var err error
	if useWatch {
		watchRep, err = goruntime.CollectWatch(ctx, base, interval, samples)
	} else {
		singleRep, err = goruntime.Collect(base)
	}
	if err != nil {
		return err
	}

	uploadURL := strings.TrimSpace(opts.UploadURL)
	if uploadURL != "" {
		var payload any
		if watchRep != nil {
			payload = watchRep
		} else {
			payload = &goruntime.WatchReport{
				GeneratedAt:   singleRep.GeneratedAt,
				Target:        singleRep.Target,
				SampleCount:   1,
				Samples:       []*goruntime.Report{singleRep},
				TrendFindings: nil,
			}
		}
		if err := postRuntimeWatchSample(ctx, uploadURL, opts.SessionID, opts.SampleToken, payload); err != nil {
			return err
		}
	}

	jsonOut := opts.JSON || strings.EqualFold(outputFormat, "json")
	if useWatch {
		if jsonOut {
			return goruntime.WriteWatchJSON(os.Stdout, watchRep)
		}
		return writeWatchText(os.Stdout, watchRep)
	}
	if jsonOut {
		return goruntime.WriteJSON(os.Stdout, singleRep)
	}
	return goruntime.WriteText(os.Stdout, singleRep)
}

func writeWatchText(w io.Writer, wr *goruntime.WatchReport) error {
	if wr == nil {
		return nil
	}
	fmt.Fprintf(w, "Go Runtime 观测（多采样）\n")
	fmt.Fprintf(w, "样本数: %d  间隔: %.0fs\n", wr.SampleCount, wr.IntervalSeconds)
	if len(wr.TrendFindings) > 0 {
		fmt.Fprintf(w, "\n趋势发现:\n")
		for i, f := range wr.TrendFindings {
			fmt.Fprintf(w, "%d. [%s] %s\n", i+1, strings.ToUpper(f.Severity), f.Title)
			fmt.Fprintf(w, "   证据: %s\n", f.Evidence)
		}
	}
	if len(wr.Samples) > 0 {
		fmt.Fprintf(w, "\n最近一次快照:\n")
		return goruntime.WriteText(w, wr.Samples[len(wr.Samples)-1])
	}
	return nil
}

func bindGoRuntimeFlags(cmd *cobra.Command, opts *goRuntimeCLIOptions) {
	cmd.Flags().IntVar(&opts.PID, "pid", 0, "本地宿主机进程 PID（与 --pod 二选一）")
	cmd.Flags().StringVarP(&opts.Namespace, "namespace", "n", "", "Kubernetes namespace（配合 --pod）")
	cmd.Flags().StringVar(&opts.Pod, "pod", "", "Kubernetes Pod 名称；须在 Pod 所在节点执行，且已安装 crictl")
	cmd.Flags().StringVar(&opts.Container, "container", "", "容器名称（默认可省略，取 Pod 内第一个容器）")
	cmd.Flags().StringVar(&opts.ProcRoot, "proc-root", "/proc", "procfs 根目录")
	cmd.Flags().StringVar(&opts.CgroupRoot, "cgroup-root", "/sys/fs/cgroup", "cgroupfs 根目录")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "输出机器可读 JSON")
	cmd.Flags().DurationVar(&opts.WatchInterval, "watch-interval", 0, "多采样间隔（例如 10s）；--watch-samples>1 时未设置则默认 10s")
	cmd.Flags().IntVar(&opts.WatchSamples, "watch-samples", 1, "采集次数；>1 时启用间隔采样并生成趋势结论")
	cmd.Flags().StringVar(&opts.UploadURL, "upload-url", "", "将本次观测 JSON POST 到平台（例如 https://host/ft-api/api/runtime-watch/sample）")
	cmd.Flags().StringVar(&opts.SessionID, "session-id", "", "与 --upload-url 配套：会话 UUID")
	cmd.Flags().StringVar(&opts.SampleToken, "sample-token", "", "与 --upload-url 配套：控制台创建会话时给出的写入令牌")
	cmd.Flags().StringVar(&opts.CrictlPath, "crictl-path", "", "crictl 可执行文件路径，默认可从 PATH 查找")
}

func bindGoRuntimeAnalyzeFlags(cmd *cobra.Command, opts *goRuntimeCLIOptions) {
	// namespace / pod / -n 已由 analyze 父命令注册，RunE 中从包级变量写入 opts。
	cmd.Flags().IntVar(&opts.PID, "pid", 0, "Go runtime: 本地宿主机进程 PID")
	cmd.Flags().StringVar(&opts.Container, "container", "", "Go runtime: Kubernetes container 名称")
	cmd.Flags().StringVar(&opts.ProcRoot, "proc-root", "/proc", "Go runtime: procfs 根目录")
	cmd.Flags().StringVar(&opts.CgroupRoot, "cgroup-root", "/sys/fs/cgroup", "Go runtime: cgroupfs 根目录")
	cmd.Flags().DurationVar(&opts.WatchInterval, "watch-interval", 0, "Go runtime: 多采样间隔")
	cmd.Flags().IntVar(&opts.WatchSamples, "watch-samples", 1, "Go runtime: 采集次数")
	cmd.Flags().StringVar(&opts.UploadURL, "upload-url", "", "Go runtime: 上报样本 URL")
	cmd.Flags().StringVar(&opts.SessionID, "session-id", "", "Go runtime: 会话 id")
	cmd.Flags().StringVar(&opts.SampleToken, "sample-token", "", "Go runtime: 样本写入 token")
	cmd.Flags().StringVar(&opts.CrictlPath, "crictl-path", "", "Go runtime: crictl 路径")
}
