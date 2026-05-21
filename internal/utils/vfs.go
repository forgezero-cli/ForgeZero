package utils

import (
	"sync"

	fzvfs "fz/internal/fs"
)

var (
	vfsMu sync.RWMutex
	vfs   fzvfs.FileSystem = fzvfs.Default
)

func SetFileSystem(f fzvfs.FileSystem) {
	vfsMu.Lock()
	defer vfsMu.Unlock()
	if f == nil {
		vfs = fzvfs.Default
		return
	}
	vfs = f
}

func fileSystem() fzvfs.FileSystem {
	vfsMu.RLock()
	defer vfsMu.RUnlock()
	return vfs
}

func ResetFileSystem() {
	SetFileSystem(nil)
}
