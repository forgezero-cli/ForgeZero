// SPDX-License-Identifier: MIT

package auto

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildToolCreatesAndProducesBinaryInTempProject(t *testing.T) {
	// FIXME:
	t.Skip("temporarily skip due to changes -mode raw; will fix later")

	if runtime.GOOS == "windows" {
		t.Skip("test targets unix-like toolchain")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "main.c")
	bin := filepath.Join(dir, "app")
	content := strings.Join([]string{
		"int main(){return 0;}",
	}, "\n")
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/fz", "-cc", src, "-out", bin, "-mode", "raw", "-format", "elf64", "-no-cache", "-no-sanitize")
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	stdoutStderr := &bytes.Buffer{}
	cmd.Stdout = stdoutStderr
	cmd.Stderr = stdoutStderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("fz build failed: %v\n%s", err, stdoutStderr.String())
	}

	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("expected binary %s to be created: %vfz output:\n%s", bin, err, stdoutStderr.String())
	}
}
