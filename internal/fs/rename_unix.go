//go:build !windows

package fs

import "os"

func renameAtomic(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
