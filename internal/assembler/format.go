package assembler

import (
	"strings"
)

func IsBinFormat() bool {
	return OutputFormat == "bin"
}

func IsBareMetalTarget() bool {
	t := Target
	if strings.Contains(t, "baremetal") || strings.Contains(t, "bare-metal") {
		return true
	}
	if strings.Contains(t, "cortex-") {
		return true
	}
	if strings.Contains(t, "none-") {
		return true
	}
	if strings.Contains(t, "unknown-elf") {
		return true
	}
	if strings.Contains(t, "-elf") && !strings.Contains(t, "linux") {
		return true
	}
	return false
}

func SkipLinker() bool {
	return IsBinFormat()
}

func formatFlagForTarget() string {
	if IsBinFormat() {
		return "-fbin"
	}
	switch {
	case isWasmTarget():
		return ""
	case strings.Contains(Target, "x86_64"):
		return "-felf64"
	case strings.Contains(Target, "i386") || strings.Contains(Target, "i686"):
		return "-felf32"
	case strings.Contains(Target, "arm"):
		return "-march=armv7-a"
	default:
		return "-felf64"
	}
}
