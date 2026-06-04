package assembler

import (
	"flag"
	"runtime"
	"strings"
)

func DefaultTargetFromGOARCH() string {
	switch runtime.GOARCH {
	case "amd64":
		if runtime.GOOS == "windows" {
			return "x86_64-windows-gnu"
		}
		return "x86_64-linux-gnu"
	case "386":
		if runtime.GOOS == "windows" {
			return "i686-windows-gnu"
		}
		return "i386-linux-gnu"
	case "arm64":
		return "aarch64-linux-gnu"
	case "arm":
		return "arm-linux-gnueabihf"
	case "riscv64":
		return "riscv64-unknown-elf"
	default:
		return "x86_64-linux-gnu"
	}
}

func ApplyDefaultTarget() {
	if muslFlag := flag.Lookup("musl"); muslFlag != nil && muslFlag.Value.String() != "" {
		return
	}
	if Target == "" || Target == "x86_64-linux-gnu" {
		Target = DefaultTargetFromGOARCH()
	}
}

func TargetFromGOARCHExplicit() string {
	return DefaultTargetFromGOARCH()
}

func NormalizeTargetTriple(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return DefaultTargetFromGOARCH()
	}
	return t
}
