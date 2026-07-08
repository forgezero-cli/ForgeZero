//go:build linux
// +build linux

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
 *   along with this program.  If not, see <https://www.gnu.org/licenses/\>.
 */

package builder

import (
	"syscall"

	"github.com/forgezero-cli/ForgeZero/internal/logger"
)

func mmapFile(fd int, size int) ([]byte, error) {
	logger.Debug("MAP_POPULATE enabled for mmap\n")
	return syscall.Mmap(fd, 0, size, syscall.PROT_READ, syscall.MAP_SHARED|syscall.MAP_POPULATE)
}

func munmapFile(data []byte) error {
	return syscall.Munmap(data)
}

func prefetchMappedFile(data []byte) {
	if len(data) > 0 {
		_ = syscall.Madvise(data, syscall.MADV_WILLNEED)
	}
}
