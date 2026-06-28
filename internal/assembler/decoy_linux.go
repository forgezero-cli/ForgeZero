//go:build linux

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

package assembler

import (
	"os"
	"syscall"
)

func emitDecoyObject(path string) error {
	const size = 4096
	buf := make([]byte, size)
	copy(buf[:4], []byte{0x7f, 'E', 'L', 'F'})
	for i := 4; i < size; i++ {
		buf[i] = byte((i * 31) ^ 0xAA)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC|syscall.O_CLOEXEC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Truncate(int64(size)); err != nil {
		return err
	}
	data, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return err
	}
	copy(data, buf)
	return syscall.Munmap(data)
}
