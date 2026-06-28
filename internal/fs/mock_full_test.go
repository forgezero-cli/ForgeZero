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

package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMockAllOpsSuccess(t *testing.T) {
	dir := t.TempDir()
	m := NewMock(Default)
	path := filepath.Join(dir, "f")
	if err := m.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := m.WriteFile(path, []byte("d"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ReadFile(path); err != nil {
		t.Fatal(err)
	}
	rc, err := m.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	rc.Close()
	ov, err := m.OpenVerified(path)
	if err != nil {
		t.Fatal(err)
	}
	ov.Close()
	tmp, err := m.CreateTemp(dir, "t*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	tmpName := tmp.Name()
	tmp.Close()
	if err := m.Chmod(tmpName, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := m.Rename(tmpName, path+".renamed"); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove(path + ".renamed"); err != nil {
		t.Fatal(err)
	}
	if err := m.WriteFile(path, []byte("d"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Stat(path); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Lstat(path); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ReadDir(dir); err != nil {
		t.Fatal(err)
	}
	s1, _ := m.Stat(path)
	s2, _ := m.Stat(path)
	if !m.SameFile(s1, s2) {
		t.Fatal("same file expected")
	}
}

func TestMockAllOpsFail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x")
	cases := []struct {
		op  string
		run func(*Mock) error
	}{
		{"MkdirAll", func(m *Mock) error { return m.MkdirAll(path, 0o700) }},
		{"WriteFile", func(m *Mock) error { return m.WriteFile(path, nil, 0o600) }},
		{"ReadFile", func(m *Mock) error { _, e := m.ReadFile(path); return e }},
		{"Open", func(m *Mock) error { _, e := m.Open(path); return e }},
		{"OpenVerified", func(m *Mock) error { _, e := m.OpenVerified(path); return e }},
		{"CreateTemp", func(m *Mock) error { _, e := m.CreateTemp(dir, "p"); return e }},
		{"Remove", func(m *Mock) error { return m.Remove(path) }},
		{"RemoveAll", func(m *Mock) error { return m.RemoveAll(path) }},
		{"Rename", func(m *Mock) error { return m.Rename(path, path+"2") }},
		{"Stat", func(m *Mock) error { _, e := m.Stat(path); return e }},
		{"Lstat", func(m *Mock) error { _, e := m.Lstat(path); return e }},
		{"ReadDir", func(m *Mock) error { _, e := m.ReadDir(dir); return e }},
		{"Chmod", func(m *Mock) error { return m.Chmod(path, 0o600) }},
		{"Readlink", func(m *Mock) error { _, e := m.Readlink(path); return e }},
		{"EvalSymlinks", func(m *Mock) error { _, e := m.EvalSymlinks(path); return e }},
	}
	for _, tc := range cases {
		m := NewMock(Default)
		m.SetFailOp(tc.op, ErrPermission)
		if err := tc.run(m); err != ErrPermission {
			t.Fatalf("%s: got %v", tc.op, err)
		}
	}
}

func TestMockSetFailPath(t *testing.T) {
	m := NewMock(Default)
	m.SetFail("Open", "/x", ErrTimeout)
	if m.err("Open", "/x") != ErrTimeout {
		t.Fatal("path fail")
	}
}

func TestDefaultReadlinkEval(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "t")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "l")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink")
	}
	if _, err := Default.Readlink(link); err != nil {
		t.Fatal(err)
	}
	if _, err := Default.EvalSymlinks(link); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultRemoveAll(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := Default.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := Default.RemoveAll(sub); err != nil {
		t.Fatal(err)
	}
}

func TestOpenVerifiedNotExist(t *testing.T) {
	_, err := Default.OpenVerified(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected error")
	}
}
