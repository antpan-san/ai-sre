package go_runtime

import (
	"strings"
	"testing"
)

func TestAnalyzeTargetPodDetail_Restarts(t *testing.T) {
	ref := PodRef{Namespace: "memleak-demo", Pod: "demo"}
	d := kubectlPodDetail{}
	d.Status.Phase = "Running"
	d.Spec.NodeName = "node-1"
	d.Status.ContainerStatuses = []struct {
		Name         string `json:"name"`
		Ready        bool   `json:"ready"`
		RestartCount int32  `json:"restartCount"`
		ContainerID  string `json:"containerID"`
		State        struct {
			Running    *struct{ StartedAt string `json:"startedAt"` } `json:"running"`
			Waiting    *struct{ Reason, Message string }            `json:"waiting"`
			Terminated *struct {
				Reason   string `json:"reason"`
				Message  string `json:"message"`
				ExitCode int32  `json:"exitCode"`
			} `json:"terminated"`
		} `json:"state"`
		LastState struct {
			Terminated *struct {
				Reason   string `json:"reason"`
				Message  string `json:"message"`
				ExitCode int32  `json:"exitCode"`
			} `json:"terminated"`
		} `json:"lastState"`
	}{
		{Name: "app", Ready: true, RestartCount: 5, ContainerID: "containerd://abc123"},
	}
	findings := analyzeTargetPodDetail(d, ref)
	if len(findings) == 0 {
		t.Fatal("expected findings for high restart count")
	}
	found := false
	for _, f := range findings {
		if f.Severity == severityCrit && f.Title != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected critical restart finding, got %+v", findings)
	}
}

func TestWaitingReasonDiagnosis(t *testing.T) {
	_, cause := waitingReasonDiagnosis("ImagePullBackOff")
	if !strings.Contains(cause, "镜像") {
		t.Fatalf("ImagePullBackOff cause: %s", cause)
	}
	_, cause = waitingReasonDiagnosis("CrashLoopBackOff")
	if !strings.Contains(cause, "崩溃") {
		t.Fatalf("CrashLoopBackOff cause: %s", cause)
	}
}

func TestSummarizeInfrastructureReport(t *testing.T) {
	wr := BuildInfrastructureWatchReport(
		PodRef{Namespace: "ns", Pod: "p", Container: "c"},
		kubectlPodDetail{},
		"id", "col", "kube-system", "busybox:1.36",
		errTest("ImagePullBackOff"),
		"Failed to pull image",
	)
	sum := SummarizeInfrastructureReport(wr)
	if sum.Level == "" || sum.Title == "" {
		t.Fatalf("unexpected summary: %+v", sum)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }
