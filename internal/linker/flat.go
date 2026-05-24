package linker

import (
	"context"
	"syscall"

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
	sfd, err := syscall.Open(src, syscall.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(sfd)
	var st syscall.Stat_t
	if err := syscall.Fstat(sfd, &st); err != nil {
		return err
	}
	mode := uint32(st.Mode & 0777)
	dfd, err := syscall.Open(dst, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer syscall.Close(dfd)
	var buf [65536]byte
	for {
		rn, rerr := syscall.Read(sfd, buf[:])
		if rn > 0 {
			written := 0
			for written < rn {
				wn, werr := syscall.Write(dfd, buf[written:rn])
				if werr != nil {
					return werr
				}
				written += wn
			}
		}
		if rerr != nil {
			if rerr == syscall.EINTR {
				continue
			}
			return rerr
		}
		if rn == 0 {
			break
		}
	}
	return nil
}

func shouldSkipLinker() bool {
	return assembler.SkipLinker()
}
