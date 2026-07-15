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

package utils

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
)

var (
	ErrScanOpen     = errors.New("scan: open")
	ErrScanMmap     = errors.New("scan: mmap")
	ErrScanResolve  = errors.New("scan: resolve")
	includeBytes    = []byte("include")
	warnOutsideHead = []byte("WARNING: include outside root ignored: ")
)

func hashPath(path string) uint64 {
	h := uint64(1469598103934665603)
	for i := 0; i < len(path); i++ {
		h ^= uint64(path[i])
		h *= 1099511628211
	}
	return h
}

func resolveIncludePathBytes(currentDir string, include []byte) (string, error) {
	var tmp [4096]byte
	n := copy(tmp[:], currentDir)
	if n > 0 && tmp[n-1] != os.PathSeparator {
		tmp[n] = os.PathSeparator
		n++
	}
	m := copy(tmp[n:], include)
	resolved := filepath.Clean(string(tmp[:n+m]))
	return ResolveSecurePathCached(resolved)
}

func warnOutsideRoot(path string) {
	var tmp [4096]byte
	n := copy(tmp[:], warnOutsideHead)
	if n+len(path)+1 > len(tmp) {
		os.Stderr.Write(warnOutsideHead)
		os.Stderr.Write([]byte(path))
		os.Stderr.Write([]byte{'\n'})
		return
	}
	n += copy(tmp[n:], path)
	tmp[n] = '\n'
	os.Stderr.Write(tmp[:n+1])
}

func mmapPath(path string) ([]byte, bool, error) {
	resolved, err := ResolveSecurePathCached(path)
	if err != nil {
		return nil, false, ErrScanResolve
	}
	f, err := openVerified(resolved)
	if err != nil {
		return nil, false, ErrScanOpen
	}
	of, ok := f.(interface {
		Stat() (os.FileInfo, error)
		Fd() uintptr
	})
	if !ok {
		f.Close()
		return nil, false, ErrScanOpen
	}
	fi, err := of.Stat()
	if err != nil {
		f.Close()
		return nil, false, err
	}
	if fi.Size() == 0 {
		f.Close()
		return nil, false, nil
	}
	if fi.Size() < 64*1024 {
		f.Close()
		data, err := fileSystem().ReadFile(resolved)
		if err != nil {
			return nil, false, ErrScanOpen
		}
		return data, false, nil
	}
	data, err := mmapFile(getFileDescriptor(of), fi.Size())
	f.Close()
	if err != nil {
		return nil, false, ErrScanMmap
	}
	return data, true, nil
}

func scanFileIncludes(path string, buf []string) ([]string, error) {
	data, mmapped, err := mmapPath(path)
	if err != nil {
		return buf, err
	}
	if len(data) == 0 {
		return buf, nil
	}
	if mmapped {
		defer func() { _ = unmapFile(data) }()
	}

	currentDir := filepath.Dir(path)
	pos := 0
	for {
		idx := bytes.Index(data[pos:], includeBytes)
		if idx == -1 {
			break
		}
		start := pos + idx
		hashPos := start
		if hashPos > 0 && data[hashPos-1] != '#' {
			pos = start + len(includeBytes)
			continue
		}
		i := hashPos - 1
		for i > 0 && (data[i] == ' ' || data[i] == '\t') {
			i--
		}
		if i < 0 || data[i] != '#' {
			pos = start + len(includeBytes)
			continue
		}
		i = start + len(includeBytes)
		for i < len(data) && (data[i] == ' ' || data[i] == '\t') {
			i++
		}
		if i >= len(data) {
			pos = len(data)
			break
		}
		switch data[i] {
		case '"':
			begin := i + 1
			end := begin
			for end < len(data) && data[end] != '"' {
				end++
			}
			if end >= len(data) {
				pos = len(data)
				break
			}
			resolved, err := resolveIncludePathBytes(currentDir, data[begin:end])
			if err == nil {
				buf = append(buf, resolved)
			}
			pos = end + 1
		case '<':
			begin := i + 1
			end := begin
			for end < len(data) && data[end] != '>' {
				end++
			}
			if end >= len(data) {
				pos = len(data)
				break
			}
			resolved, err := resolveIncludePathBytes(currentDir, data[begin:end])
			if err == nil {
				buf = append(buf, resolved)
			}
			pos = end + 1
		default:
			pos = start + len(includeBytes)
		}
	}
	return buf, nil
}

func ScanDependencies(path string) ([]string, error) {
	resolved, err := ResolveSecurePathCached(path)
	if err != nil {
		return nil, err
	}
	return ScanDependenciesRoot(resolved, filepath.Dir(resolved))
}

func ScanDependenciesRoot(path, rootDir string) ([]string, error) {
	resolved, err := ResolveSecurePathCached(path)
	if err != nil {
		return nil, err
	}
	rootDir = filepath.Clean(rootDir)
	stack := []string{resolved}
	visited := make(map[uint64]struct{}, 64)
	deps := make([]string, 0, 64)
	incBuffer := make([]string, 0, 16)

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		h := hashPath(cur)
		if _, ok := visited[h]; ok {
			continue
		}
		visited[h] = struct{}{}
		deps = append(deps, cur)

		incBuffer = incBuffer[:0]
		includes, err := scanFileIncludes(cur, incBuffer)
		if err != nil {
			return nil, err
		}
		for _, inc := range includes {
			if !pathWithinRoot(rootDir, inc) {
				parent := filepath.Dir(rootDir)
				if parent == rootDir || !pathWithinRoot(parent, inc) {
					warnOutsideRoot(inc)
					continue
				}
			}
			hi := hashPath(inc)
			if _, ok := visited[hi]; ok {
				continue
			}
			stack = append(stack, inc)
		}
	}
	return deps, nil
}
