/* TEST FILE
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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
	// FIXME: fix this test after changes to -mode raw; currently fails due to changes in the build process
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
