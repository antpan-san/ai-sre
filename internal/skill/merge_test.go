package skill

import "testing"

func TestMergeRegistriesOverride(t *testing.T) {
	a := &Registry{Packs: []Pack{{Name: "x", DisplayName: "A"}}}
	b := &Registry{Packs: []Pack{{Name: "x", DisplayName: "B"}}}
	m := MergeRegistries(a, b)
	if len(m.Packs) != 1 || m.Packs[0].DisplayName != "B" {
		t.Fatalf("%+v", m.Packs)
	}
}
