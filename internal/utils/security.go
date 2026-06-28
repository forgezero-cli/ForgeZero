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
	"runtime"
	"strings"
)

const (
	DirPerm  os.FileMode = 0o700
	FilePerm os.FileMode = 0o600
)

func forbiddenPathChars() string {
	if runtime.GOOS == "windows" {
		return "`$&|;><*?[]{}()\"'\x00\n\r"
	}
	return "`$&|;><*?[]{}()\"'\\\x00\n\r"
}

func forbiddenArgChars() string {
	if runtime.GOOS == "windows" {
		return "`$&|;><*?[]{}()\"'\x00\n\r"
	}
	return "`$&|;><*?[]{}()\"'\\\x00\n\r"
}

func pathWithinRoot(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if filepath.VolumeName(root) != "" && filepath.VolumeName(root) != filepath.VolumeName(target) {
		return false
	}
	if strings.EqualFold(root, target) {
		return true
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func isUnsafeUNC(path string) bool {
	if !strings.HasPrefix(path, `\\`) {
		return false
	}
	rest := strings.TrimPrefix(path, `\\`)
	parts := strings.SplitN(rest, `\`, 3)
	return len(parts) < 2 || parts[0] == "" || parts[1] == ""
}
