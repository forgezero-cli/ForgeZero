//go:build !linux || !amd64
// +build !linux !amd64

package io_uring

import "os"

func Enabled() bool { return false }

func initRing() error {
	return os.ErrInvalid
}

func ReadFile(path string) ([]byte, error) {
	return nil, os.ErrInvalid
}

func WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.ErrInvalid
}
