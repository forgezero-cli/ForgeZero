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
	"os"
	"path/filepath"
	"testing"

	fzvfs "github.com/forgezero-cli/ForgeZero/internal/fs"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func TestScanTreeOpenFail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("x"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	m := fzvfs.NewMock(fzvfs.Default)
	resolved, err := utils.ResolveSecurePath(path)
	if err != nil {
		t.Fatal(err)
	}
	m.SetFail("OpenVerified", resolved, fzvfs.ErrPermission)
	utils.SetFileSystem(m)
	t.Cleanup(func() { utils.SetFileSystem(nil) })
	_, _, err = scanTree(dir)
	if err == nil {
		t.Fatal("expected open error")
	}
}

func TestHealthyFromChecksPermFail(t *testing.T) {
	r := Report{
		Healthy:     true,
		Toolchain:   []ToolCheck{{Name: "zig", Required: false, Found: true}},
		Permissions: PermReport{Readable: false, Writable: false, Error: "denied"},
	}
	r.healthyFromChecks()
	if r.Healthy {
		t.Fatal("expected unhealthy")
	}
}

func TestAuditPermissionsStatFail(t *testing.T) {
	dir := t.TempDir()
	m := fzvfs.NewMock(fzvfs.Default)
	resolved, _ := utils.ResolveSecurePath(dir)
	m.SetFail("Stat", resolved, fzvfs.ErrPermission)
	utils.SetFileSystem(m)
	t.Cleanup(func() { utils.SetFileSystem(nil) })
	pr := auditPermissions(dir)
	if pr.Readable || pr.Writable {
		t.Fatalf("got %+v", pr)
	}
}

func TestRunFasmOptionalOnWindows(t *testing.T) {
	tools := auditToolchain()
	for _, tc := range tools {
		if tc.Name == "fasm" && tc.Required {
			if os.Getenv("GOOS") == "windows" {
				t.Fatal("fasm should not be required on windows")
			}
		}
	}
	_ = context.Background()
}

func TestProbeWritableRemoveFail(t *testing.T) {
	dir := t.TempDir()
	if err := probeWritable(dir); err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(dir, ".fz_doctor_probe")
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFail("Remove", probe, fzvfs.ErrPermission)
	utils.SetFileSystem(m)
	t.Cleanup(func() { utils.SetFileSystem(nil) })
	if err := probeWritable(dir); err == nil {
		t.Fatal("expected remove error")
	}
}
