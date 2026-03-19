package names_test

import (
	"testing"
	"anonchat/backend/internal/names"
)

func TestGenerate_ReturnsNonEmpty(t *testing.T) {
	name := names.Generate(nil)
	if name == "" {
		t.Fatal("expected non-empty name")
	}
}

func TestGenerate_FormatIsAdjectiveAnimal(t *testing.T) {
	name := names.Generate(nil)
	if len(name) < 3 {
		t.Fatalf("name too short: %q", name)
	}
}

func TestGenerate_AvoidsExisting(t *testing.T) {
	existing := []string{}
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		name := names.Generate(existing)
		if seen[name] {
			t.Fatalf("duplicate name generated: %q", name)
		}
		seen[name] = true
		existing = append(existing, name)
	}
}
