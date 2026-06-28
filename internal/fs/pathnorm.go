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
	"path/filepath"
	"strings"
)

func CleanPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, `\\`) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.FromSlash(path))
}

func IsUNC(path string) bool {
	return len(path) >= 2 && (path[0] == '\\' && path[1] == '\\')
}

func HasDrivePrefix(path string) bool {
	if len(path) < 2 {
		return false
	}
	return path[1] == ':' && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z'))
}

func NormalizeAbs(path string) (string, error) {
	clean := CleanPath(path)
	if IsUNC(clean) {
		return clean, nil
	}
	return filepath.Abs(clean)
}
