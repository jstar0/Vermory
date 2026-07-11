package brand

import "testing"

func TestIdentity(t *testing.T) {
	if Name != "Vermory" {
		t.Fatalf("expected product name Vermory, got %q", Name)
	}
	if Slug != "vermory" {
		t.Fatalf("expected product slug vermory, got %q", Slug)
	}
	if Tagline != "Governed Memory for AI" {
		t.Fatalf("unexpected tagline %q", Tagline)
	}
}
