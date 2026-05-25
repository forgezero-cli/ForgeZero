//go:build !linux && !windows
// +build !linux,!windows

package linker

import (
	"context"
	"io"
	"os"
)

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

func copyFileHot(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	return err
}

func unlinkHot(path string) error {
	return os.Remove(path)
}
