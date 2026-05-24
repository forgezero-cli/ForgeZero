//go:build !windows
// +build !windows

package linker

import (
	"context"
	"syscall"
	"unsafe"

	"fz/internal/assembler"
	"fz/internal/utils"
)

func linkFlatBinary(ctx context.Context, obj, bin string) error {
	if obj == bin {
		return nil
	}
	if err := utils.EnsureDir(bin); err != nil {
		return err
	}
	return copyFileHot(obj, bin)
}

func copyFileHot(src, dst string) error {
	sfd, err := openHot(src, syscall.O_RDONLY, 0)
	if err != nil {
		return err
	}
	var st syscall.Stat_t
	if err := syscall.Fstat(sfd, &st); err != nil {
		_ = syscall.Close(sfd)
		return err
	}
	mode := uint32(st.Mode & 0777)
	dfd, err := openHot(dst, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, mode)
	if err != nil {
		_ = syscall.Close(sfd)
		return err
	}
	var buf [65536]byte
	for {
		rn, rerr := syscall.Read(sfd, buf[:])
		if rn > 0 {
			written := 0
			for written < rn {
				wn, werr := syscall.Write(dfd, buf[written:rn])
				if werr != nil {
					_ = syscall.Close(dfd)
					_ = syscall.Close(sfd)
					return werr
				}
				written += wn
			}
		}
		if rerr != nil {
			if rerr == syscall.EINTR {
				continue
			}
			_ = syscall.Close(dfd)
			_ = syscall.Close(sfd)
			return rerr
		}
		if rn == 0 {
			break
		}
	}
	if err := syscall.Close(dfd); err != nil {
		_ = syscall.Close(sfd)
		return err
	}
	if err := syscall.Close(sfd); err != nil {
		return err
	}
	return nil
}

const atFDCWD = ^uintptr(99)

func openHot(path string, flags int, perm uint32) (int, error) {
	var buf [4096]byte
	n := copy(buf[:], path)
	if n >= len(buf) {
		return -1, syscall.ENAMETOOLONG
	}
	buf[n] = 0
	fd, _, errno := syscall.Syscall6(syscall.SYS_OPENAT, atFDCWD, uintptr(unsafe.Pointer(&buf[0])), uintptr(flags), uintptr(perm), 0, 0)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

func unlinkHot(path string) error {
	var buf [4096]byte
	n := copy(buf[:], path)
	if n >= len(buf) {
		return syscall.ENAMETOOLONG
	}
	buf[n] = 0
	_, _, errno := syscall.Syscall(syscall.SYS_UNLINKAT, atFDCWD, uintptr(unsafe.Pointer(&buf[0])), 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func shouldSkipLinker() bool {
	return assembler.SkipLinker()
}
