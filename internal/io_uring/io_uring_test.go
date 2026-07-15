//go:build linux && amd64
// +build linux,amd64

/*
 *   Copyright (c) 2026 ForgeZero-cli
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
	"os"
	"path/filepath"
	"testing"
	"unsafe"
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

func TestIoUringStructSizes(t *testing.T) {
	t.Logf("ioUringSqe size = %d", unsafe.Sizeof(ioUringSqe{}))
	t.Logf("ioUringSqringOffsets size = %d", unsafe.Sizeof(ioUringSqringOffsets{}))
	t.Logf("ioUringCqringOffsets size = %d", unsafe.Sizeof(ioUringCqringOffsets{}))
	t.Logf("ioUringParams size = %d", unsafe.Sizeof(ioUringParams{}))
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
