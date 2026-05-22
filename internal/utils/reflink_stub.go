//go:build !linux

package utils

import "os"

func LinkOrClone(src, dst string) error {
    if err := os.Link(src, dst); err == nil {
        return nil
    }
    return CopyFile(src, dst)
}
