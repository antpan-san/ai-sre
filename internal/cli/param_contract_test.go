package cli

import (
	"strings"
	"testing"
)

func TestValidateParamContractUnknownCommand(t *testing.T) {
	root := newRoot("ai-sre")
	res := ValidateParamContract(root, []string{"chekc", "kafka"})
	if res == nil || res.OK {
		t.Fatal("expected param contract failure")
	}
	if res.AutoIterationCreated {
		t.Fatal("param errors must not create auto iteration")
	}
	if res.Layer != "param" {
		t.Fatalf("layer=%q", res.Layer)
	}
	if len(res.Suggestions) == 0 {
		t.Fatal("expected suggestions")
	}
	if len(res.Suggestions) > 3 {
		t.Fatalf("too many suggestions: %d", len(res.Suggestions))
	}
}

func TestValidateParamContractExemptDoctor(t *testing.T) {
	root := newRoot("ai-sre")
	if res := ValidateParamContract(root, []string{"doctor"}); res != nil {
		t.Fatalf("doctor should be exempt: %+v", res)
	}
}

func TestBuildCommandCatalogDigestStable(t *testing.T) {
	root := newRoot("ai-sre")
	d1 := CommandCatalogDigest(root)
	d2 := CommandCatalogDigest(root)
	if d1 == "" || d1 != d2 {
		t.Fatalf("digest unstable: %q %q", d1, d2)
	}
}

func TestSuggestCommandsPreferCheck(t *testing.T) {
	root := newRoot("ai-sre")
	cat := BuildCommandCatalog(root)
	sugs := suggestCommands(cat, "chekc", 3)
	if len(sugs) == 0 {
		t.Fatal("no suggestions")
	}
	found := false
	for _, s := range sugs {
		if strings.HasPrefix(s.Value, "check") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected check* suggestion, got %+v", sugs)
	}
}
