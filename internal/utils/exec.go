package utils

import (
	"os/exec"
	"runtime"
	"strings"
)

func lookExecutable(name string) (string, error) {
	if runtime.GOOS != "windows" {
		return exec.LookPath(name)
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".bat") || strings.HasSuffix(lower, ".cmd") {
		return exec.LookPath(name)
	}
	if p, err := exec.LookPath(name + ".exe"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath(name + ".bat"); err == nil {
		return p, nil
	}
	return exec.LookPath(name)
}
