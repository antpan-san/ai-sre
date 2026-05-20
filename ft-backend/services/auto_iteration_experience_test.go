package services

import (
	"ft-backend/models"
	"testing"
)

func TestShouldRecordAutoIterationExperience(t *testing.T) {
	if !shouldRecordAutoIterationExperience(true, models.AutoIterationStatusCompleted) {
		t.Fatal("completed success should record")
	}
	if shouldRecordAutoIterationExperience(false, models.AutoIterationStatusCompleted) {
		t.Fatal("failed should not record")
	}
	if !isSkillAccumulationSource(models.AutoIterationSourceSkillRefine) {
		t.Fatal("skill_refine is accumulation source")
	}
}
