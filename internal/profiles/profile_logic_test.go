package profiles

import "testing"

func TestParseUserProfile_DefaultBalanced(t *testing.T) {
	p := ParseUserProfile("")
	if p.Name != "balanced" {
		t.Fatalf("expected balanced, got %q", p.Name)
	}
}

func TestParseUserProfile_NormalizesCaseAndAliases(t *testing.T) {
	p := ParseUserProfile("PoWeReD")
	if p.Name != "performance" {
		t.Fatalf("expected performance, got %q", p.Name)
	}

	p = ParseUserProfile("eco")
	if p.Name != "power-saver" {
		t.Fatalf("expected power-saver, got %q", p.Name)
	}
}

func TestOptimizationFlag(t *testing.T) {
	if ParseUserProfile("balanced").OptimizationFlag() != "-O2" {
		t.Fatalf("balanced optimization flag mismatch")
	}
	if ParseUserProfile("performance").OptimizationFlag() != "-O3" {
		t.Fatalf("performance optimization flag mismatch")
	}
	if ParseUserProfile("power-saver").OptimizationFlag() != "-Os" {
		t.Fatalf("power-saver optimization flag mismatch")
	}
}

func TestEffectiveJobs_UsesRequestedJobsWhenPositive(t *testing.T) {
	jobs := ParseUserProfile("performance").EffectiveJobs(8)
	if jobs != 8 {
		t.Fatalf("expected requested jobs to be preserved, got %d", jobs)
	}
}

func TestDefaultJobs_IsAtLeastOne(t *testing.T) {
	p := ParseUserProfile("power-saver")
	if p.DefaultJobs() != 1 {
		t.Fatalf("expected power-saver DefaultJobs=1, got %d", p.DefaultJobs())
	}
}
