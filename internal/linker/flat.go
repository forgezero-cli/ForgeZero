package linker

import (
	"context"
	"os"
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
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()
	st, err := sf.Stat()
	if err != nil {
		return err
	}
	df, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, utils.FilePerm)
	if err != nil {
		return err
	}
	defer df.Close()
	var buf [65536]byte
	remain := st.Size()
	for remain > 0 {
		n := int(remain)
		if n > len(buf) {
			n = len(buf)
		}
		rn, rerr := sf.Read(buf[:n])
		if rerr != nil {
			return rerr
		}
		wn := 0
		for wn < rn {
			m, werr := syscall.Write(int(df.Fd()), buf[wn:rn])
			if werr != nil {
				return werr
			}
			wn += m
		}
		remain -= int64(rn)
	}
	return nil
}

func shouldSkipLinker() bool {
	return assembler.SkipLinker()
}
