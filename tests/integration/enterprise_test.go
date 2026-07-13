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

package integration_test

import (
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/forgezero-cli/ForgeZero/internal/builder"
	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func writeDummySource(dir string) (string, error) {
	p := filepath.Join(dir, "main.asm")
	data := []byte("section .text\nglobal _start\n_start:\n    mov eax, 60\n    xor edi, edi\n    syscall\n")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return "", err
	}
	return p, nil
}

func buildMockAssembler(t *testing.T, destPath string) {
	t.Helper()
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "main.go")
	src := `package main

import (
	"bufio"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]
	
	var expanded []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			f, err := os.Open(arg[1:])
			if err == nil {
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line != "" {
						expanded = append(expanded, line)
					}
				}
				f.Close()
			}
		} else {
			expanded = append(expanded, arg)
		}
	}
	
	out := ""
	for i, arg := range expanded {
		if arg == "-o" && i+1 < len(expanded) {
			out = expanded[i+1]
			break
		}
	}
	
	if out != "" {
		elfHeader := []byte{
			0x7f, 'E', 'L', 'F', 2, 1, 1, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			1, 0, 0x3e, 0, 1, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 64, 0, 0, 0,
			0, 0, 0, 0, 64, 0, 0, 0,
		}
		os.WriteFile(out, elfHeader, 0644)
	}
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatalf("write mock source: %v", err)
	}
	
	cmd := exec.Command("go", "build", "-o", destPath, srcPath)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile mock tool: %v\n%s", err, out)
	}
}

func buildMockLinker(t *testing.T, destPath string) {
	t.Helper()
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "main.go")
	src := `package main

import (
	"bufio"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]
	
	var expanded []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			f, err := os.Open(arg[1:])
			if err == nil {
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line != "" {
						expanded = append(expanded, line)
					}
				}
				f.Close()
			}
		} else {
			expanded = append(expanded, arg)
		}
	}
	
	out := ""
	for i, arg := range expanded {
		if arg == "-o" && i+1 < len(expanded) {
			out = expanded[i+1]
			break
		}
	}
	
	if out != "" {
		elfHeader := []byte{
			0x7f, 'E', 'L', 'F', 2, 1, 1, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			2, 0, 0x3e, 0, 1, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 64, 0, 0, 0,
			0, 0, 0, 0, 64, 0, 0, 0,
		}
		os.WriteFile(out, elfHeader, 0644)
	}
}
`
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatalf("write mock source: %v", err)
	}
	
	cmd := exec.Command("go", "build", "-o", destPath, srcPath)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile mock tool: %v\n%s", err, out)
	}
}

func TestEnterpriseIsolation(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go compiler not available")
	}
	
	dir := t.TempDir()
	_, err := writeDummySource(dir)
	if err != nil {
		t.Fatalf("write source: %v", err)
	}

	cfg := &config.Config{}
	cfg.Isolation = config.IsolationStandard
	cfg.DeterministicStrip = true
	cfg.ToolchainSettings.SearchPriority = []string{"local", "system"}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ctx = utils.ContextWithConfig(ctx, cfg)

	toolBin := filepath.Join(dir, "toolchain", "bin")
	if err := os.MkdirAll(toolBin, 0o755); err != nil {
		t.Fatalf("mkdir toolchain: %v", err)
	}
	nasmPath := filepath.Join(toolBin, "nasm")
	ldPath := filepath.Join(toolBin, "ld")
	
	buildMockAssembler(t, nasmPath)
	buildMockLinker(t, ldPath)

	utils.SetExecutionRoot(dir)
	os.Setenv("FZ_TEST_VARIANT", "A")
	res1, err := builder.BuildDir(ctx, []string{dir}, filepath.Join(dir, "out1"), false, false, "raw", false, true, false, true, false, nil, nil, nil, nil, nil, 1, "executable")
	if err != nil {
		t.Fatalf("build1 failed: %v", err)
	}
	b1, err := os.ReadFile(res1.Binary)
	if err != nil {
		t.Fatalf("read bin1: %v", err)
	}
	h1 := sha256.Sum256(b1)

	os.Setenv("FZ_TEST_VARIANT", "B")
	res2, err := builder.BuildDir(ctx, []string{dir}, filepath.Join(dir, "out2"), false, false, "raw", false, true, false, true, false, nil, nil, nil, nil, nil, 1, "executable")
	if err != nil {
		t.Fatalf("build2 failed: %v", err)
	}
	b2, err := os.ReadFile(res2.Binary)
	if err != nil {
		t.Fatalf("read bin2: %v", err)
	}
	h2 := sha256.Sum256(b2)

	if h1 != h2 {
		t.Fatalf("binaries differ under isolation: %x vs %x", h1, h2)
	}
}