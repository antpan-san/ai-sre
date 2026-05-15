package go_runtime

import "testing"

func TestDeriveLocalDiagnosis_CollectorImagePull(t *testing.T) {
	wr := &WatchReport{
		TrendFindings: []Finding{{
			Severity: severityCrit,
			Title:    "采集器镜像拉取失败",
			Evidence: "Back-off pulling image busybox:1.36",
			Cause:    "离线集群无法拉取默认 busybox 镜像",
		}},
		ProbeBundle: map[string]string{
			"kubectl_collector_describe": "Warning Failed ... ImagePullBackOff",
		},
	}
	s, ok := DeriveLocalDiagnosis(wr)
	if !ok {
		t.Fatal("expected local diagnosis")
	}
	if s.Title == "" || s.Evidence == "" {
		t.Fatalf("unexpected summary: %+v", s)
	}
}

func TestApplyDiagnosis(t *testing.T) {
	wr := &WatchReport{}
	ApplyDiagnosis(wr, "CRITICAL", "内存持续增长", "rss 4x in 30s", "local")
	if wr.Diagnosis.RootCause != "内存持续增长" {
		t.Fatalf("diagnosis: %+v", wr.Diagnosis)
	}
	if wr.Summary.Title != wr.Diagnosis.RootCause {
		t.Fatalf("summary: %+v", wr.Summary)
	}
}
