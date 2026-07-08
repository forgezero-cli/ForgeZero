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
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package builder

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

type ioUringParams struct {
	sqEntries    uint32
	cqEntries    uint32
	flags        uint32
	wqFd         int32
	resv         [3]uint32
	sqOff        ioUringSqringOffsets
	cqOff        ioUringCqringOffsets
}

type ioUringSqringOffsets struct {
	off                   uint32
	head                  uint32
	tail                  uint32
	sqRingMask            uint32
	sqEntries             uint32
	flags                 uint32
	oeFlags               uint32
	reserved              uint64
}

type ioUringCqringOffsets struct {
	off                   uint32
	head                  uint32
	tail                  uint32
	cqRingMask            uint32
	nent                  uint32
	overflow              uint32
	cqes                  uint64
}

func IsIOUringSupported() bool {
	var params ioUringParams
	fd, _, err := unix.Syscall(unix.SYS_IO_URING_SETUP, uintptr(1), uintptr(unsafe.Pointer(&params)), 0)
	if int(fd) < 0 {
		return false
	}
	_ = unix.Close(int(fd))
	return err == 0
}
