//go:build !linux

package utils

import (
    "os"
    "path/filepath"
)

func Walk(root string, fn func(path string, info os.FileInfo, err error) error) error {
    return filepath.Walk(root, fn)
}
