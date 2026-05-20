package services

import (
	"testing"
	"time"
)

func TestProcessSkillFeedbackUnhelpfulThreshold(t *testing.T) {
	t.Parallel()
	reg, err := NewSkillRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	topic := "redis"
	for i := 0; i < 3; i++ {
		v := false
		_ = reg.AppendFeedback(SkillFeedback{Time: time.Now().UTC(), Topic: topic, Helpful: &v})
	}
	v := false
	eval, err := ProcessSkillFeedback(reg, SkillFeedback{Time: time.Now().UTC(), Topic: topic, Helpful: &v, Note: "结论不对"})
	if err != nil {
		t.Fatal(err)
	}
	if eval.UnhelpfulCount < 3 {
		t.Fatalf("unhelpful=%d", eval.UnhelpfulCount)
	}
	if !eval.ReviewTriggered {
		t.Fatal("expected review triggered at threshold")
	}
}

func TestUpdateEnhancementReviewStatus(t *testing.T) {
	t.Parallel()
	reg, err := NewSkillRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := UpdateEnhancementReviewStatus(reg, "req-1", "", "redis", "refined", "ok"); err != nil {
		t.Fatal(err)
	}
	rows, err := ListEnhancementReviews(reg, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 || rows[0].EnhancementStatus != "refined" {
		t.Fatalf("rows=%+v", rows)
	}
}
