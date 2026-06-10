package profiles

import "testing"

func TestParseUserProfile_DefaultBalanced_EmptyInput(t *testing.T) {
	p := ParseUserProfile("")
	if p.Name != "balanced" {
		t.Fatalf("expected balanced, got %q", p.Name)
	}
}

func TestParseUserProfile_NormalizesCase_SeparateFile(t *testing.T) {
	p := ParseUserProfile("PoWeReD")
	if p.Name != "performance" {
		t.Fatalf("expected performance, got %q", p.Name)
	}
}
