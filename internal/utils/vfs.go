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
	"sync/atomic"

	fzvfs "github.com/forgezero-cli/ForgeZero/internal/fs"
)

type vfsHolder struct{ fs fzvfs.FileSystem }

var vfs atomic.Value

func init() { vfs.Store(vfsHolder{fs: fzvfs.Default}) }

func SetFileSystem(f fzvfs.FileSystem) {
	if f == nil {
		vfs.Store(vfsHolder{fs: fzvfs.Default})
		return
	}
	vfs.Store(vfsHolder{fs: f})
}

func fileSystem() fzvfs.FileSystem { return vfs.Load().(vfsHolder).fs }

func ResetFileSystem() { SetFileSystem(nil) }
