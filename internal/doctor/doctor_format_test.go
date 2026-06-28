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

package doctor

import (
	"context"
	"strings"
	"testing"
)

func TestFormatHumanFull(t *testing.T) {
	r := Report{
		Status: "degraded",
		Platform: PlatformReport{
			GOOS: "linux", GOARCH: "amd64", FileSystemImpl: "unix",
			PathSeparator: "/", ExecutionRoot: "/proj", NumCPU: 4,
		},
		Toolchain: []ToolCheck{
			{Name: "gcc", Required: true, Found: true, Path: "/usr/bin/gcc"},
			{Name: "fasm", Required: false, Found: false},
		},
		Permissions: PermReport{
			Root: "/proj", Readable: true, Writable: false,
			DirsScanned: 2, FilesSeen: 5, Error: "partial",
		},
		Errors: []string{"tool missing"},
	}
	out := FormatHuman(r)
	for _, sub := range []string{"degraded", "toolchain:", "gcc", "permissions:", "partial", "tool missing"} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in %s", sub, out)
		}
	}
}

func TestRunEmptyProject(t *testing.T) {
	dir := t.TempDir()
	r, err := Run(context.Background(), Options{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status == "" {
		t.Fatal("empty status")
	}
}
