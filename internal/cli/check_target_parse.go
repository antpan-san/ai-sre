package cli

import (
	"fmt"
	"strconv"
	"strings"
)

func applyUnifiedCheckTarget(ctx map[string]string, topic string, args []string, goOpts *goRuntimeCLIOptions) error {
	t := normalizeCheckTopicAlias(topic)
	explicit := ""
	if len(args) >= 2 {
		explicit = strings.TrimSpace(args[1])
	}
	switch t {
	case "go":
		if goOpts == nil {
			return fmt.Errorf("internal: go runtime options missing")
		}
		return applyGoCheckTarget(explicit, goOpts)
	case "k8s":
		applyK8sCheckTarget(explicit, ctx)
		return nil
	case "code":
		if explicit == "" {
			return fmt.Errorf("check code 需要错误码，例: check code OPSFLEET_K8S_E_PAUSE_MISSING")
		}
		if ctx != nil {
			ctx["error_code"] = strings.ToUpper(explicit)
		}
		return nil
	case "linux":
		if explicit != "" && !strings.EqualFold(explicit, "localhost") {
			if ctx != nil {
				ctx["host"] = explicit
			}
		}
		applyCheckTargetContext(ctx, topic, args)
		return nil
	case "domain":
		mergeDomainIntoContext(ctx, topic, args)
		return nil
	default:
		applyCheckTargetContext(ctx, topic, args)
		return nil
	}
}

func applyGoCheckTarget(target string, opts *goRuntimeCLIOptions) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("go 排查需要目标，例: check go pid/1234 | check go pod/default/api-0 | check go name/my-service")
	}
	parts := strings.SplitN(target, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return fmt.Errorf("go 目标格式: pid/<pid> | name/<name> | pod/<namespace>/<pod> | pod/<pod>")
	}
	kind := strings.ToLower(strings.TrimSpace(parts[0]))
	rest := strings.TrimSpace(parts[1])
	switch kind {
	case "pid":
		pid, err := strconv.Atoi(rest)
		if err != nil || pid <= 0 {
			return fmt.Errorf("无效 pid %q", rest)
		}
		opts.PID = pid
	case "name":
		opts.PIDName = rest
	case "pod":
		opts.PodTarget = rest
	default:
		return fmt.Errorf("未知 go 目标类型 %q，支持 pid/name/pod", kind)
	}
	return nil
}

func applyK8sCheckTarget(target string, ctx map[string]string) {
	if ctx == nil {
		return
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return
	}
	lower := strings.ToLower(target)
	switch lower {
	case "pending", "crashloop", "instability":
		ctx["issue"] = lower
		ctx["pod"] = lower
		return
	}
	parts := strings.Split(target, "/")
	if len(parts) < 2 {
		ctx["pod"] = target
		return
	}
	switch strings.ToLower(parts[0]) {
	case "pod":
		if len(parts) == 2 {
			ctx["pod"] = parts[1]
		} else {
			ctx["namespace"] = parts[1]
			ctx["pod"] = parts[2]
		}
	case "deployment", "statefulset", "daemonset", "replicaset", "job", "cronjob", "service", "ingress", "pvc":
		ctx[parts[0]] = strings.Join(parts[1:], "/")
	default:
		ctx["pod"] = target
	}
}

func validateCheckTargetForTopic(topic, target string) error {
	if err := validateCheckTargetLiteral(target); err != nil {
		return err
	}
	t := normalizeCheckTopicAlias(topic)
	if t == "go" {
		var opts goRuntimeCLIOptions
		return applyGoCheckTarget(target, &opts)
	}
	return nil
}

func checkTargetDisplay(topic string, args []string, ctx map[string]string) string {
	if len(args) >= 2 && strings.TrimSpace(args[1]) != "" {
		return strings.TrimSpace(args[1])
	}
	t := normalizeCheckTopicAlias(topic)
	if spec, ok := checkTargetSpecs[t]; ok && ctx != nil {
		if v := strings.TrimSpace(ctx[spec.PrimaryKey]); v != "" {
			return v
		}
	}
	if t == "linux" {
		return "localhost"
	}
	return ""
}
