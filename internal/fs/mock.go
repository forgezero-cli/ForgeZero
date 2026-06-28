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
	"io"
	"os"
	"sync"
)

type Mock struct {
	Base     FileSystem
	mu       sync.Mutex
	failures map[string]error
}

func NewMock(base FileSystem) *Mock {
	if base == nil {
		base = Default
	}
	return &Mock{Base: base, failures: make(map[string]error)}
}

func (m *Mock) SetFail(op, path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failures[op+":"+path] = err
}

func (m *Mock) SetFailOp(op string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failures[op+":"] = err
}

func (m *Mock) err(op, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.failures[op+":"+path]; ok {
		return e
	}
	if e, ok := m.failures[op+":"]; ok {
		return e
	}
	return nil
}

func (m *Mock) MkdirAll(path string, perm os.FileMode) error {
	if e := m.err("MkdirAll", path); e != nil {
		return e
	}
	return m.Base.MkdirAll(path, perm)
}

func (m *Mock) WriteFile(path string, data []byte, perm os.FileMode) error {
	if e := m.err("WriteFile", path); e != nil {
		return e
	}
	return m.Base.WriteFile(path, data, perm)
}

func (m *Mock) ReadFile(path string) ([]byte, error) {
	if e := m.err("ReadFile", path); e != nil {
		return nil, e
	}
	return m.Base.ReadFile(path)
}

func (m *Mock) Open(path string) (io.ReadCloser, error) {
	if e := m.err("Open", path); e != nil {
		return nil, e
	}
	return m.Base.Open(path)
}

func (m *Mock) OpenVerified(path string) (io.ReadCloser, error) {
	if e := m.err("OpenVerified", path); e != nil {
		return nil, e
	}
	return m.Base.OpenVerified(path)
}

func (m *Mock) CreateTemp(dir, pattern string) (*os.File, error) {
	if e := m.err("CreateTemp", dir); e != nil {
		return nil, e
	}
	return m.Base.CreateTemp(dir, pattern)
}

func (m *Mock) Remove(name string) error {
	if e := m.err("Remove", name); e != nil {
		return e
	}
	return m.Base.Remove(name)
}

func (m *Mock) RemoveAll(path string) error {
	if e := m.err("RemoveAll", path); e != nil {
		return e
	}
	return m.Base.RemoveAll(path)
}

func (m *Mock) Rename(oldpath, newpath string) error {
	if e := m.err("Rename", oldpath); e != nil {
		return e
	}
	return m.Base.Rename(oldpath, newpath)
}

func (m *Mock) Stat(name string) (os.FileInfo, error) {
	if e := m.err("Stat", name); e != nil {
		return nil, e
	}
	return m.Base.Stat(name)
}

func (m *Mock) Lstat(name string) (os.FileInfo, error) {
	if e := m.err("Lstat", name); e != nil {
		return nil, e
	}
	return m.Base.Lstat(name)
}

func (m *Mock) ReadDir(name string) ([]os.DirEntry, error) {
	if e := m.err("ReadDir", name); e != nil {
		return nil, e
	}
	return m.Base.ReadDir(name)
}

func (m *Mock) Chmod(name string, mode os.FileMode) error {
	if e := m.err("Chmod", name); e != nil {
		return e
	}
	return m.Base.Chmod(name, mode)
}

func (m *Mock) Readlink(name string) (string, error) {
	if e := m.err("Readlink", name); e != nil {
		return "", e
	}
	return m.Base.Readlink(name)
}

func (m *Mock) EvalSymlinks(path string) (string, error) {
	if e := m.err("EvalSymlinks", path); e != nil {
		return "", e
	}
	return m.Base.EvalSymlinks(path)
}

func (m *Mock) SameFile(a, b os.FileInfo) bool {
	return m.Base.SameFile(a, b)
}
