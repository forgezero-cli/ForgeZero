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

package gloria

import (
	"os"
	"strconv"
	"testing"
	"unsafe"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
	"golang.org/x/sys/unix"
)

func TestKernelPokeAndPeek(t *testing.T) {
	src, err := os.ReadFile("test_kernel.glo")
	if err != nil {
		t.Fatal("read test_kernel.glo: " + err.Error())
	}
	if _, err = Emit(string(src)); err != nil {
		t.Fatal("compile test_kernel: " + err.Error())
	}

	pageSize := os.Getpagesize()

	fd, err := unix.MemfdCreate("forgezero", unix.MFD_CLOEXEC|unix.MFD_EXEC)
	if err != nil {
		t.Fatal("memfd_create: " + err.Error())
	}
	defer unix.Close(fd)
	if err := unix.Ftruncate(fd, int64(pageSize)); err != nil {
		t.Fatal("ftruncate: " + err.Error())
	}
	execArea, err := unix.Mmap(fd, 0, pageSize,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_SHARED)
	if err != nil {
		t.Fatal("mmap exec: " + err.Error())
	}
	defer func() {
		if err := unix.Munmap(execArea); err != nil {
			t.Fatal("failed to munmap: " + err.Error())
		}
	}()

	dataArea, err := unix.Mmap(-1, 0, pageSize,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_ANON|unix.MAP_PRIVATE|0x40) // 0x40 = MAP_32BIT
	if err != nil {
		t.Fatal("mmap data: " + err.Error())
	}
	defer func() {
		if err := unix.Munmap(dataArea); err != nil {
			t.Fatal("failed to munmap: " + err.Error())
		}
	}()

	videoMemAddr := &dataArea[0]
	*(*uint16)(unsafe.Pointer(videoMemAddr)) = 0x0F41

	addrStr := strconv.Itoa(int(uintptr(unsafe.Pointer(videoMemAddr))))
	program := `fn main() {
    let screen = ` + addrStr + `;
    let original_char = peek(screen);
    poke(screen, 2631);
    let new_char = peek(screen);
    return new_char;
}`

	machineCode, err := Emit(program)
	if err != nil {
		t.Fatal("compile error: " + err.Error())
	}
	copy(execArea, machineCode)

	if err := unix.Mprotect(execArea, unix.PROT_READ|unix.PROT_EXEC); err != nil {
		t.Fatal("mprotect execArea: " + err.Error())
	}

	ret := utils.ExecRawRet(machineCode)
	if ret != 2631 {
		t.Errorf("expected 2631, got %d", ret)
	}
}
