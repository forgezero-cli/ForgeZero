package assembler

import (
	"os/exec"
	"testing"
)

func TestCcCxxGasForAllTargets(t *testing.T) {
	old := Target
	defer func() { Target = old }()
	cases := []struct {
		target  string
		wantCC  string
		wantCXX string
		wantGas string
		wantFmt string
	}{
		{"x86_64-linux-gnu", "gcc", "g++", "as", "-felf64"},
		{"i386-linux-gnu", "gcc", "g++", "as", "-felf32"},
		{"arm-linux-gnueabihf", "arm-linux-gnueabihf-gcc", "arm-linux-gnueabihf-g++", "arm-linux-gnueabihf-as", "-march=armv7-a"},
		{"riscv64-unknown-elf", "riscv64-unknown-elf-gcc", "riscv64-unknown-elf-g++", "riscv64-unknown-elf-as", "-felf64"},
		{"wasm32-unknown-unknown", "clang", "clang++", "clang", ""},
	}
	for _, tc := range cases {
		Target = tc.target
		if tc.target == "wasm32-unknown-unknown" {
			if _, err := exec.LookPath("emcc"); err == nil {
				tc.wantCC = "emcc"
			}
			if _, err := exec.LookPath("em++"); err == nil {
				tc.wantCXX = "em++"
			}
		}
		if got := ccForTarget(); got != tc.wantCC {
			t.Fatalf("%s cc: %s", tc.target, got)
		}
		if got := cxxForTarget(); got != tc.wantCXX {
			t.Fatalf("%s cxx: %s", tc.target, got)
		}
		if got := gasCmdForTarget(); got != tc.wantGas {
			t.Fatalf("%s gas: %s", tc.target, got)
		}
		if got := formatFlagForTarget(); got != tc.wantFmt {
			t.Fatalf("%s fmt: %s", tc.target, got)
		}
	}
}
