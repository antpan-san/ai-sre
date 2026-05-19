package services

import (
	"testing"

	"ft-backend/models"
)

func TestAssessFulfillmentRiskHighForMigration(t *testing.T) {
	risk, high := assessFulfillmentRisk("ai-sre deploy", "product_gap", "needs db migration", SkillExecutionIntent{Topic: "billing"})
	if !high || risk != models.AutoIterationRiskHigh {
		t.Fatalf("risk=%s high=%v", risk, high)
	}
}

func TestAssessFulfillmentRiskLowForCapabilityGap(t *testing.T) {
	risk, high := assessFulfillmentRisk("ai-sre check postgresql", "capability_gap", "", SkillExecutionIntent{Topic: "postgresql"})
	if high || risk != models.AutoIterationRiskLow {
		t.Fatalf("risk=%s high=%v", risk, high)
	}
}

func TestPublicFulfillmentMessageStripsSecrets(t *testing.T) {
	msg := publicFulfillmentMessage("entitlement yaml prompt leaked", "safe")
	if msg != "safe" {
		t.Fatalf("msg=%q", msg)
	}
}
