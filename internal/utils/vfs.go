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
	"sync"

	fzvfs "github.com/forgezero-cli/ForgeZero/internal/fs"
)

var (
	vfsMu sync.RWMutex
	vfs   fzvfs.FileSystem = fzvfs.Default
)

func SetFileSystem(f fzvfs.FileSystem) {
	vfsMu.Lock()
	defer vfsMu.Unlock()
	if f == nil {
		vfs = fzvfs.Default
		return
	}
	vfs = f
}

func fileSystem() fzvfs.FileSystem {
	vfsMu.RLock()
	defer vfsMu.RUnlock()
	return vfs
}

func ResetFileSystem() {
	SetFileSystem(nil)
}
