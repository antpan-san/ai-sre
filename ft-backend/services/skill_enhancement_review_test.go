package services

import (
	"strings"
	"testing"
	"time"
)

func TestEvaluateSkillEnhancement_Delegation(t *testing.T) {
	in := PostAICallRecord{
		Topic:        "redis",
		CommandKind:  "analyze",
		SkillName:    "redis_diagnose_v1",
		MatchedSkill: true,
		Answer:       "请执行 redis-cli INFO memory 确认碎片率",
		UserContext:  map[string]string{"redis_diagnose_json": `{"memory":{}}`},
		EvidenceKeys: []string{"redis_diagnose_json"},
	}
	r := EvaluateSkillEnhancement(nil, in)
	if !r.NeedsEnhancement {
		t.Fatal("expected needs_enhancement")
	}
	if r.Priority == "" {
		t.Fatal("expected priority")
	}
	found := false
	for _, rec := range r.Recommendations {
		if strings.Contains(rec, "手工执行") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected delegation recommendation, got %v", r.Recommendations)
	}
}

func TestEvaluateSkillEnhancement_MissingEvidence(t *testing.T) {
	in := PostAICallRecord{
		Topic:        "linux",
		CommandKind:  "analyze",
		MatchedSkill: true,
		Answer:       "根因是 IO 等待",
	}
	r := EvaluateSkillEnhancement(nil, in)
	if !r.NeedsEnhancement {
		t.Fatal("expected needs_enhancement for missing linux evidence")
	}
}

func TestUpdateEnhancementReviewStatusByReviewKey(t *testing.T) {
	t.Parallel()
	reg, err := NewSkillRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	openAt := time.Date(2026, 5, 20, 5, 9, 8, 520107735, time.UTC)
	open := SkillEnhancementReview{
		Time:             openAt,
		Topic:            "redis",
		CommandKind:      "engagement",
		NeedsEnhancement: true,
		Priority:         "high",
		SavingsScore:     70,
		EnhancementStatus: "open",
	}
	if err := appendEnhancementReviewLog(reg, open); err != nil {
		t.Fatal(err)
	}
	list, err := ListEnhancementReviews(reg, 10, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 open review, got %d", len(list))
	}
	key := list[0].ReviewKey
	if key == "" {
		t.Fatal("expected review_key on listed row")
	}
	if err := UpdateEnhancementReviewStatus(reg, "", key, "redis", "refined", "ok"); err != nil {
		t.Fatal(err)
	}
	list, err = ListEnhancementReviews(reg, 10, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected open list empty after refine, got %+v", list)
	}
}

func TestListEnhancementReviews(t *testing.T) {
	dir := t.TempDir()
	reg := &SkillRegistry{dataDir: dir}
	r1 := SkillEnhancementReview{
		Time:             time.Now().UTC(),
		Topic:            "redis",
		NeedsEnhancement: true,
		Priority:         "high",
		SavingsScore:     70,
		EnhancementStatus: "open",
	}
	if err := appendEnhancementReviewLog(reg, r1); err != nil {
		t.Fatal(err)
	}
	list, err := ListEnhancementReviews(reg, 10, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 review, got %d", len(list))
	}
	sum, err := SummarizeEnhancementReviews(reg, 5)
	if err != nil {
		t.Fatal(err)
	}
	if sum.OpenCount != 1 || sum.HighPriority != 1 {
		t.Fatalf("unexpected summary: %+v", sum)
	}
}
