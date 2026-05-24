package linker

import "testing"

func TestTargetProfileForKnownBareMetal(t *testing.T) {
	profile, ok := TargetProfileFor("baremetal")
	if !ok {
		t.Fatal("expected baremetal profile")
	}
	if profile.Name != "baremetal" {
		t.Fatalf("expected profile.Name baremetal, got %q", profile.Name)
	}
}

func TestIsBareMetalTarget(t *testing.T) {
	old := Target
	defer func() { Target = old }()

	SetTarget("baremetal")
	if !IsBareMetalTarget() {
		t.Fatal("baremetal target should be recognized as bare metal")
	}

	SetTarget("x86_64-linux-gnu")
	if IsBareMetalTarget() {
		t.Fatal("linux target should not be recognized as bare metal")
	}
}
