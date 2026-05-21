//go:build windows

package fs

import (
	"os"
	"time"
)

func renameAtomic(oldpath, newpath string) error {
	var last error
	for attempt := 0; attempt < 8; attempt++ {
		if err := os.Rename(oldpath, newpath); err == nil {
			return nil
		} else {
			last = err
			time.Sleep(time.Millisecond * time.Duration(10*(attempt+1)))
		}
	}
	return last
}
