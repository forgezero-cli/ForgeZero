//go:build linux && amd64
// +build linux,amd64

/*
 *   Copyright (c) 2026 forgezero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package io_uring

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func TestReadWriteFileViaIoUring(t *testing.T) {
	if os.Getenv("FORGEZERO_IO_URING") != "1" {
		t.Skip("io_uring disabled")
	}
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "io_uring_test.dat")
	data := []byte("forgezero-io-uring-test")
	if err := WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	read, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(read) != string(data) {
		t.Fatalf("unexpected data: got %q want %q", read, data)
	}
}

func TestReadFilesViaIoUring(t *testing.T) {
	if os.Getenv("FORGEZERO_IO_URING") != "1" {
		t.Skip("io_uring disabled")
	}
	tmpDir := t.TempDir()
	paths := []string{filepath.Join(tmpDir, "one"), filepath.Join(tmpDir, "empty"), filepath.Join(tmpDir, "two")}
	if err := os.WriteFile(paths[0], []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths[1], nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths[2], []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := ReadFiles(paths)
	if err != nil {
		t.Fatal(err)
	}
	if string(data[0]) != "one" || len(data[1]) != 0 || string(data[2]) != "two" {
		t.Fatalf("unexpected batch data: %q %q %q", data[0], data[1], data[2])
	}
}

func TestReadFilesFallback(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "fallback")
	if err := os.WriteFile(path, []byte("fallback"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FORGEZERO_IO_URING", "0")
	data, err := ReadFiles([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 || string(data[0]) != "fallback" {
		t.Fatalf("unexpected fallback data: %q", data)
	}
}

func TestStatxAtFallback(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "statx")
	if err := os.WriteFile(path, []byte("statx"), 0o644); err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Open(tmpDir, unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	stat, err := StatxAt(fd, "statx", unix.AT_SYMLINK_NOFOLLOW, unix.STATX_TYPE|unix.STATX_SIZE|unix.STATX_INO)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Ino == 0 || stat.Size != 5 {
		t.Fatalf("unexpected statx result: inode=%d size=%d", stat.Ino, stat.Size)
	}
}

func TestIoUringStructSizes(t *testing.T) {
	t.Logf("ioUringSqe size = %d", unsafe.Sizeof(ioUringSqe{}))
	t.Logf("ioUringSqringOffsets size = %d", unsafe.Sizeof(ioUringSqringOffsets{}))
	t.Logf("ioUringCqringOffsets size = %d", unsafe.Sizeof(ioUringCqringOffsets{}))
	t.Logf("ioUringParams size = %d", unsafe.Sizeof(ioUringParams{}))
}

func TestValidateResult(t *testing.T) {
	tests := []struct {
		name     string
		result   *ioUringCqe
		expected int
		shortErr error
		wantErr  error
	}{
		{"nil completion", nil, 4, io.ErrUnexpectedEOF, os.ErrInvalid},
		{"short read", &ioUringCqe{res: 3}, 4, io.ErrUnexpectedEOF, io.ErrUnexpectedEOF},
		{"short write", &ioUringCqe{res: 3}, 4, io.ErrShortWrite, io.ErrShortWrite},
		{"complete", &ioUringCqe{res: 4}, 4, io.ErrUnexpectedEOF, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validateResult(test.result, test.expected, test.shortErr); got != test.wantErr {
				t.Fatalf("validateResult() = %v, want %v", got, test.wantErr)
			}
		})
	}
}

func TestIoUringRuntimeState(t *testing.T) {
	if os.Getenv("FORGEZERO_IO_URING") != "1" {
		t.Skip("io_uring disabled")
	}
	if !Enabled() {
		t.Fatal("expected io_uring enabled")
	}
	t.Logf("ringFd=%d, enabled=%v", ringFd, enabled)
	t.Logf("sqEntries=%d, cqEntries=%d", *sqRingEntries, *cqRingEntries)
	t.Logf("sqRingMask=%d, cqRingMask=%d", *sqRingMask, *cqRingMask)
	t.Logf("sqTail=%d, sqHead=%d", *sqTail, *sqHead)
	t.Logf("cqTail=%d, cqHead=%d", *cqTail, *cqHead)
	t.Logf("sqes len=%d", len(sqes))
	t.Logf("cqes len=%d", len(cqes))
	if len(sqes) == 0 || len(cqes) == 0 {
		t.Fatal("empty SQE or CQE slice")
	}
}
