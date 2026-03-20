package ui_test

import (
	"testing"

	"anonchat/tui/internal/ui"
)

func TestUserColor_Deterministic(t *testing.T) {
	c1 := ui.UserColor("Blue Fox")
	c2 := ui.UserColor("Blue Fox")
	if c1 != c2 {
		t.Fatalf("expected same color, got %q and %q", c1, c2)
	}
}

func TestUserColor_NonEmpty(t *testing.T) {
	color := ui.UserColor("Blue Fox")
	if color == "" {
		t.Fatal("expected non-empty color")
	}
}
