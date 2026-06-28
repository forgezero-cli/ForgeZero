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

package linker

import (
	"context"
	"io"
	"os"
	"syscall"
)

func copyFileHot(src, dst string) error {
	srcPtr, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	dstPtr, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	sfd, err := syscall.CreateFile(srcPtr, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return err
	}
	dfd, err := syscall.CreateFile(dstPtr, syscall.GENERIC_WRITE, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil, syscall.CREATE_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		_ = syscall.CloseHandle(sfd)
		return err
	}
	var buf [65536]byte
	for {
		var rn uint32
		err = syscall.ReadFile(sfd, buf[:], &rn, nil)
		if err != nil {
			if err == syscall.ERROR_HANDLE_EOF {
				if rn == 0 {
					break
				}
			} else {
				_ = syscall.CloseHandle(dfd)
				_ = syscall.CloseHandle(sfd)
				return err
			}
		}
		if rn == 0 {
			break
		}
		written := uint32(0)
		for written < rn {
			var wn uint32
			err = syscall.WriteFile(dfd, buf[written:rn], &wn, nil)
			if err != nil {
				_ = syscall.CloseHandle(dfd)
				_ = syscall.CloseHandle(sfd)
				return err
			}
			written += wn
		}
	}
	if err := syscall.CloseHandle(dfd); err != nil {
		_ = syscall.CloseHandle(sfd)
		return err
	}
	if err := syscall.CloseHandle(sfd); err != nil {
		return err
	}
	return nil
}

func unlinkHot(path string) error {
	return os.Remove(path)
}

func shouldSkipLinker() bool {
	return false
}

func linkFlatBinary(ctx context.Context, obj, bin string) error {
	sf, err := os.Open(obj)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.Create(bin)
	if err != nil {
		return err
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	return err
}
