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
	PIDName    string
	Namespace  string
	Pod        string
	PodTarget  string
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
	var smart goRuntimeCLIOptions
	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Go 程序智能诊断",
		Long: `一条命令完成 Go 程序运行时诊断。支持本机 PID、进程名或 Kubernetes Pod；
默认自动采样约 30 秒，CLI 输出结论，并把报告上传到当前绑定账号的执行记录与进程观测页面。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("请使用 --pid、--pid-name 或 --pod 指定诊断目标")
			}
			return runSmartGoRuntimeDiagnose(cmd.Context(), smart)
		},
	}
	cmd.Flags().IntVar(&smart.PID, "pid", 0, "本机 Go 进程 PID")
	cmd.Flags().StringVar(&smart.PIDName, "pid-name", "", "本机 Go 进程名或命令行关键词")
	cmd.Flags().StringVar(&smart.PodTarget, "pod", "", "Kubernetes Pod，格式: pod | namespace/pod | namespace/pod/container")
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

func runSmartGoRuntimeDiagnose(ctx context.Context, opts goRuntimeCLIOptions) error {
	targets := 0
	if opts.PID > 0 {
		targets++
	}
	if strings.TrimSpace(opts.PIDName) != "" {
		targets++
	}
	if strings.TrimSpace(opts.PodTarget) != "" {
		targets++
	}
	if targets != 1 {
		return fmt.Errorf("请且仅请提供一个诊断目标：--pid、--pid-name 或 --pod")
	}
	base := strings.TrimRight(strings.TrimSpace(resolveOpsfleetAPIBase()), "/")
	token := strings.TrimSpace(resolveOpsfleetToken())
	fingerprint := strings.TrimSpace(resolveOpsfleetFingerprint())
	if base == "" || token == "" || fingerprint == "" {
		return fmt.Errorf("Go runtime 诊断需要当前 ai-sre 已绑定用户 token；请从控制台重新生成并执行「安装 ai-sre」命令")
	}
	if err := checkGoRuntimeAuth(ctx, base, token, fingerprint); err != nil {
		return fmt.Errorf("Go runtime 诊断鉴权失败: %w", err)
	}

	const smartSamples = 4
	const smartInterval = 10 * time.Second
	var wr *goruntime.WatchReport
	var err error
	command := strings.Join(os.Args, " ")
	switch {
	case opts.PID > 0:
		wr, err = goruntime.CollectWatch(ctx, goruntime.Options{PID: opts.PID}, smartInterval, smartSamples)
		if wr != nil {
			wr.Target.Target = fmt.Sprintf("pid:%d", opts.PID)
			wr.Target.Source = "pid"
			for _, s := range wr.Samples {
				if s != nil {
					s.Target.Target = wr.Target.Target
					s.Target.Source = wr.Target.Source
				}
			}
			wr.Summary = goruntime.SummarizeWatchReport(wr)
		}
	case strings.TrimSpace(opts.PIDName) != "":
		selected, candidates, findErr := goruntime.FindProcessByName("/proc", opts.PIDName)
		if findErr != nil {
			return fmt.Errorf("未找到匹配进程 %q: %w", opts.PIDName, findErr)
		}
		wr, err = goruntime.CollectWatch(ctx, goruntime.Options{PID: selected.PID}, smartInterval, smartSamples)
		if wr != nil {
			wr.Candidates = candidates
			wr.Target.Target = "pid-name:" + strings.TrimSpace(opts.PIDName)
			wr.Target.Source = "pid-name"
			wr.Target.Exe = selected.Exe
			for _, s := range wr.Samples {
				if s != nil {
					s.Target.Target = wr.Target.Target
					s.Target.Source = wr.Target.Source
					s.Target.Exe = selected.Exe
				}
			}
			wr.Summary = goruntime.SummarizeWatchReport(wr)
		}
	case strings.TrimSpace(opts.PodTarget) != "":
		wr, err = goruntime.CollectKubernetesWatch(ctx, goruntime.KubernetesCollectOptions{
			Target:             opts.PodTarget,
			CollectorImage:     strings.TrimSpace(os.Getenv("OPSFLEET_GO_RUNTIME_COLLECTOR_IMAGE")),
			CollectorNamespace: strings.TrimSpace(os.Getenv("OPSFLEET_GO_RUNTIME_COLLECTOR_NAMESPACE")),
			KeepCollector:      strings.EqualFold(strings.TrimSpace(os.Getenv("OPSFLEET_GO_RUNTIME_KEEP_COLLECTOR")), "1"),
		}, smartInterval, smartSamples)
	}
	if err != nil {
		return err
	}
	jsonOut := strings.EqualFold(outputFormat, "json")
	if jsonOut {
		if err := goruntime.WriteWatchJSON(os.Stdout, wr); err != nil {
			return err
		}
	} else {
		if err := writeWatchText(os.Stdout, wr); err != nil {
			return err
		}
	}
	upload, err := postGoRuntimeReport(ctx, base, token, fingerprint, command, wr)
	if err != nil {
		return fmt.Errorf("诊断已完成，但页面记录未保存: %w", err)
	}
	if jsonOut {
		fmt.Fprintf(os.Stderr, "Go runtime 页面记录: execution=%s runtime_watch_session=%s\n", upload.ExecutionRecordID, upload.RuntimeWatchSessionID)
	} else {
		fmt.Fprintf(os.Stdout, "\n页面记录: 执行记录 %s；进程观测会话 %s\n", upload.ExecutionRecordID, upload.RuntimeWatchSessionID)
	}
	return nil
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
	if wr.Summary.Level != "" {
		fmt.Fprintf(w, "结论: [%s] %s\n", wr.Summary.Level, wr.Summary.Title)
		if wr.Summary.Evidence != "" {
			fmt.Fprintf(w, "证据: %s\n", wr.Summary.Evidence)
		}
		if wr.Summary.Action != "" {
			fmt.Fprintf(w, "建议: %s\n", wr.Summary.Action)
		}
	}
	if wr.Target.Target != "" {
		fmt.Fprintf(w, "目标: %s", wr.Target.Target)
		if wr.Target.PID > 0 {
			fmt.Fprintf(w, "  host_pid=%d", wr.Target.PID)
		}
		if wr.Target.Node != "" {
			fmt.Fprintf(w, "  node=%s", wr.Target.Node)
		}
		fmt.Fprintln(w)
	}
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
