//go:build windows
// +build windows

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

package audit

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	modadvapi32 = syscall.NewLazyDLL("advapi32.dll")
	procGetFileSecurityW = modadvapi32.NewProc("GetFileSecurityW")
)

const (
	OWNER_SECURITY_INFORMATION = 0x00000001
	GROUP_SECURITY_INFORMATION = 0x00000002
	DACL_SECURITY_INFORMATION  = 0x00000004
	SACL_SECURITY_INFORMATION  = 0x00000008
	LABEL_SECURITY_INFORMATION = 0x00000010
)

func RunAudit() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		return auditFile(path)
	})
}

func auditFile(path string) error {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	var secDescSize uint32
	ret, _, _ := procGetFileSecurityW.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(OWNER_SECURITY_INFORMATION|DACL_SECURITY_INFORMATION),
		0,
		0,
		uintptr(unsafe.Pointer(&secDescSize)),
	)
	if ret == 0 && secDescSize == 0 {
		return nil
	}
	secDesc := make([]byte, secDescSize)
	ret, _, _ = procGetFileSecurityW.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(OWNER_SECURITY_INFORMATION|DACL_SECURITY_INFORMATION),
		uintptr(unsafe.Pointer(&secDesc[0])),
		uintptr(secDescSize),
		uintptr(unsafe.Pointer(&secDescSize)),
	)
	if ret == 0 {
		return nil
	}
	return nil
}