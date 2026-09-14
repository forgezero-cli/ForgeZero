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

func ReadFiles(paths []string) ([][]byte, error) {
	return readFilesFallback(paths)
}

func StatxAt(dirFD int, path string, flags int, mask int) (StatxResult, error) {
	return StatxResult{}, os.ErrInvalid
}

func readFilesFallback(paths []string) ([][]byte, error) {
	data := make([][]byte, len(paths))
	for i, path := range paths {
		var err error
		data[i], err = os.ReadFile(path)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.ErrInvalid
}
