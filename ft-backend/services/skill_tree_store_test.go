package services

import (
	"testing"

	"ft-backend/models"
)

func TestBuiltinSkillTreeFallback(t *testing.T) {
	res := builtinSkillTreeFallback()
	if res.TreeRev != BuiltinSkillTreeRev {
		t.Fatalf("unexpected rev %q", res.TreeRev)
	}
	if res.Source != "builtin" {
		t.Fatalf("expected builtin source")
	}
	if len(res.Nodes) == 0 {
		t.Fatalf("expected builtin nodes")
	}
}

func TestSkillTreeNodeFromRecordStatus(t *testing.T) {
	n := skillTreeNodeFromRecord(models.SkillTreeNodeRecord{
		Path: "ops.test", Title: "T", NodeType: SkillNodeTypeCategory, Status: "disabled",
	})
	if n.Status != "disabled" {
		t.Fatalf("status=%q", n.Status)
	}
}
