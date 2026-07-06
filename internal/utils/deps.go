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
	"os"
	"path/filepath"
)

func ParseDepFile(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	i := 0
	for i < len(data) {
		if data[i] == ':' {
			i++
			break
		}
		if data[i] == '\\' && i+1 < len(data) && data[i+1] == '\n' {
			i += 2
			continue
		}
		i++
	}
	if i >= len(data) {
		return nil, nil
	}
	var deps []string
	for i < len(data) {
		for i < len(data) {
			if data[i] == '\\' && i+1 < len(data) && data[i+1] == '\n' {
				i += 2
				continue
			}
			if data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r' {
				i++
				continue
			}
			break
		}
		if i >= len(data) {
			break
		}
		start := i
		for i < len(data) {
			if data[i] == '\\' {
				if i+1 < len(data) && data[i+1] == '\n' {
					i += 2
					continue
				}
				if i+1 < len(data) && data[i+1] == ' ' {
					i += 2
					continue
				}
			}
			if data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r' {
				break
			}
			i++
		}
		token := data[start:i]
		pathBytes := unescapeDepToken(token)
		if len(pathBytes) == 0 {
			continue
		}
		deps = append(deps, string(pathBytes))
	}
	return deps, nil
}

func ParseDepFilePath(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	deps, err := ParseDepFile(data)
	if err != nil {
		return nil, err
	}
	if len(deps) == 0 {
		return nil, nil
	}
	dir := filepath.Dir(path)
	for i, dep := range deps {
		if dep == "" {
			continue
		}
		if filepath.IsAbs(dep) {
			deps[i] = filepath.Clean(dep)
			continue
		}
		deps[i] = filepath.Clean(filepath.Join(dir, dep))
	}
	return deps, nil
}

func unescapeDepToken(src []byte) []byte {
	var tmp [1024]byte
	dst := tmp[:0]
	for i := 0; i < len(src); i++ {
		if src[i] == '\\' && i+1 < len(src) {
			next := src[i+1]
			if next == '\n' {
				i++
				continue
			}
			if next == ' ' {
				dst = append(dst, ' ')
				i++
				continue
			}
		}
		dst = append(dst, src[i])
	}
	return append([]byte(nil), dst...)
}
