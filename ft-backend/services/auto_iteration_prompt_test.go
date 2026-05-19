package services

import (
	"strings"
	"testing"

	"ft-backend/models"
)

func TestFormatAutoIterationUserRequirementIncludesCompactSpec(t *testing.T) {
	desc, cmd := FormatAutoIterationUserRequirement("登录修复", "目标: 去掉误报\n验收: 登录成功", "ft-front")
	if !strings.Contains(desc, "目标: 去掉误报") {
		t.Fatalf("desc=%q", desc)
	}
	if !strings.Contains(cmd, AutoIterationDevSkillPath) {
		t.Fatalf("cmd missing dev skill path")
	}
	if !strings.Contains(cmd, "省 Token") {
		t.Fatalf("cmd missing compact spec")
	}
	if len(cmd) > 2000 {
		t.Fatalf("cmd too long: %d", len(cmd))
	}
}

func TestMergeAgentTaskMetadata(t *testing.T) {
	raw := MergeAgentTaskMetadata(models.NewJSONBFromMap(map[string]interface{}{"x": 1}))
	m := jsonbToMap(raw)
	if m["dev_spec"] != AutoIterationDevSpecVer {
		t.Fatalf("dev_spec=%v", m["dev_spec"])
	}
	if m["x"] != float64(1) && m["x"] != 1 {
		t.Fatalf("x=%v", m["x"])
	}
}
