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

package utils

import (
	"os"

	"golang.org/x/sys/unix"
)

func LinkOrClone(src, dst string) error {
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	if err := SecureMkdirAll(dst); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|unix.O_CLOEXEC, FilePerm)
	if err != nil {
		return err
	}
	if err := unix.IoctlSetInt(int(out.Fd()), unix.FICLONE, int(in.Fd())); err == nil {
		if cerr := out.Close(); cerr != nil {
			return cerr
		}
		return nil
	}
	if cerr := out.Close(); cerr != nil {
		return cerr
	}
	return CopyFile(src, dst)
}
