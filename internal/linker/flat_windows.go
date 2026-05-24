//go:build windows
// +build windows

package linker

import "syscall"

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
				// continue and write remaining bytes
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
