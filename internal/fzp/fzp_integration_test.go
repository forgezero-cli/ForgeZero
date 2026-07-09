/*
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

package fzp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFZPRealProjectBuild(t *testing.T) {
	fzPath, err := exec.LookPath("fz")
	if err != nil {
		t.Skip("fz not found in PATH, skipping integration test")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.h.in"), []byte(`#define PACKAGE_NAME "fzp_test"
#define VERSION_MAJOR 1
#define VERSION_MINOR 0
#ifdef FEATURE_X
#define HAS_FEATURE_X 1
#else
#define HAS_FEATURE_X 0
#endif
#include <stdio.h>
#if defined(__linux__)
#define PLATFORM "linux"
#elif defined(_WIN32)
#define PLATFORM "windows"
#else
#define PLATFORM "unknown"
#endif
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(`#include "config.h"
int main(void) {
    printf("package=%s version=%d.%d feature=%d platform=%s\\n", PACKAGE_NAME, VERSION_MAJOR, VERSION_MINOR, HAS_FEATURE_X, PLATFORM);
    return 0;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fz.toml"), []byte(`[preprocess]
enabled = true
inputs = ["config.h.in"]
outputs = ["config.h"]
output = "myapp"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(fzPath, "-dir", dir)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	generated := filepath.Join(dir, ".fz_objs", "include", "config.h")
	if _, err := os.Stat(generated); err != nil {
		t.Fatalf("generated config.h not found: %v", err)
	}
	binPath := ""
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Built: ") {
			binPath = strings.TrimSpace(strings.TrimPrefix(line, "Built: "))
			break
		}
	}
	if binPath == "" {
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			t.Fatalf("read project dir: %v", readErr)
		}
		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == "." || entry.Name() == ".." {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".out") || strings.HasSuffix(entry.Name(), ".exe") || entry.Name() == "app" {
				binPath = filepath.Join(dir, entry.Name())
				break
			}
		}
	}
	if binPath == "" {
		t.Fatalf("could not locate built binary from output: %s", out)
	}
	if !filepath.IsAbs(binPath) {
		binPath = filepath.Join(dir, binPath)
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("expected binary at %s: %v", binPath, err)
	}
	run := exec.Command(binPath)
	run.Dir = dir
	data, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, data)
	}
	if !strings.Contains(string(data), "package=fzp_test version=1.0 feature=0 platform=linux") {
		t.Fatalf("unexpected output: %s", data)
	}
}