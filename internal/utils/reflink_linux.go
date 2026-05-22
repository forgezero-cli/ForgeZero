//go:build linux

package utils

import (
	"golang.org/x/sys/unix"
	"os"
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
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|unix.O_CLOEXEC, FilePerm)
	if err != nil {
		return err
	}
	if err := unix.IoctlSetInt(int(out.Fd()), unix.FICLONE, int(in.Fd())); err == nil {
		out.Close()
		return nil
	}
	out.Close()
	return CopyFile(src, dst)
}
