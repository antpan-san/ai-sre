package cli

import "testing"

func TestValidateArgvInCatalogRejectsUnknownFlag(t *testing.T) {
	root := newRoot("ai-sre")
	if ValidateArgvInCatalog(root, []string{"check", "kafka", "--not-a-real-flag"}) {
		t.Fatal("expected unknown flag to fail catalog validation")
	}
}

func TestValidateArgvInCatalogAcceptsCheckKafka(t *testing.T) {
	root := newRoot("ai-sre")
	if !ValidateArgvInCatalog(root, []string{"check", "kafka"}) {
		t.Fatal("expected check kafka to pass")
	}
}

func TestFilterCatalogValidatedArgv(t *testing.T) {
	root := newRoot("ai-sre")
	in := [][]string{
		{"check", "kafka"},
		{"check", "kafka", "--bogus"},
	}
	out := FilterCatalogValidatedArgv(root, in)
	if len(out) != 1 {
		t.Fatalf("got %d want 1", len(out))
	}
}
