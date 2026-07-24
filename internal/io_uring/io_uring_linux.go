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
	"sync"
	"syscall"
	"unsafe"

	"github.com/forgezero-cli/ForgeZero/internal/logger"
	"golang.org/x/sys/unix"
)

const (
	ioUringSetupEntries    = 256
	IORING_ENTER_GETEVENTS = 1
	IORING_OFF_SQ_RING     = 0
	IORING_OFF_CQ_RING     = 0x8000000
	IORING_OFF_SQES        = 0x10000000
	IORING_OP_READ         = 22
	IORING_OP_WRITE        = 23
)

type ioUringSqringOffsets struct {
	head        uint32
	tail        uint32
	ringMask    uint32
	ringEntries uint32
	flags       uint32
	dropped     uint32
	array       uint32
	resv1       uint32
	userAddr    uint64
}

type ioUringCqringOffsets struct {
	head        uint32
	tail        uint32
	ringMask    uint32
	ringEntries uint32
	overflow    uint32
	cqes        uint32
	flags       uint32
	resv1       uint32
	userAddr    uint64
}

type ioUringParams struct {
	sqEntries    uint32
	cqEntries    uint32
	flags        uint32
	sqThreadCpu  uint32
	sqThreadIdle uint32
	features     uint32
	wqFd         uint32
	resv         [3]uint32
	sqOff        ioUringSqringOffsets
	cqOff        ioUringCqringOffsets
}

type ioUringSqe struct {
	opcode      uint8
	flags       uint8
	ioprio      uint16
	fd          int32
	off         uint64
	addr        uint64
	len         uint32
	rwFlags     uint32
	userData    uint64
	bufIndex    uint16
	personality uint16
	_pad        [20]byte
}

type ioUringCqe struct {
	res      int32
	flags    uint32
	userData uint64
}

var (
	ringFd        int
	sqRing        []byte
	cqRing        []byte
	sqesMem       []byte
	sqes          []ioUringSqe
	cqes          []ioUringCqe
	sqHead        *uint32
	sqTail        *uint32
	sqRingMask    *uint32
	sqRingEntries *uint32
	sqArray       []uint32
	cqHead        *uint32
	cqTail        *uint32
	cqRingMask    *uint32
	cqRingEntries *uint32
	mutex         sync.Mutex
	enabled       bool
	initOnce      sync.Once
)

func Enabled() bool {
	initOnce.Do(initIoUring)
	return enabled
}

func initIoUring() {
	if os.Getenv("FORGEZERO_IO_URING") != "1" {
		return
	}
	if err := initRing(); err != nil {
		return
	}
	enabled = true
}

func initRing() error {
	params := ioUringParams{sqEntries: ioUringSetupEntries, cqEntries: ioUringSetupEntries}
	fd, _, errno := unix.Syscall(unix.SYS_IO_URING_SETUP, uintptr(ioUringSetupEntries), uintptr(unsafe.Pointer(&params)), 0)
	if int(fd) < 0 {
		return errno
	}
	ringFd = int(fd)
	logger.Debug("io_uring setup succeeded\n")

	sqRingSize := int(params.sqOff.array) + int(params.sqEntries)*4
	if sqRingSize == 0 {
		_ = unix.Close(ringFd)
		return os.ErrInvalid
	}
	sq, err := unix.Mmap(ringFd, IORING_OFF_SQ_RING, sqRingSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		_ = unix.Close(ringFd)
		return err
	}

	cqRingSize := int(params.cqOff.cqes) + int(params.cqEntries)*16
	if cqRingSize == 0 {
		_ = unix.Munmap(sq)
		_ = unix.Close(ringFd)
		return os.ErrInvalid
	}
	cq, err := unix.Mmap(ringFd, IORING_OFF_CQ_RING, cqRingSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		_ = unix.Munmap(sq)
		_ = unix.Close(ringFd)
		return err
	}

	sqesSize := int(params.sqEntries) * int(unsafe.Sizeof(ioUringSqe{}))
	sqesArea, err := unix.Mmap(ringFd, IORING_OFF_SQES, sqesSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		_ = unix.Munmap(sq)
		_ = unix.Munmap(cq)
		_ = unix.Close(ringFd)
		return err
	}

	sqRing = sq
	cqRing = cq
	sqesMem = sqesArea
	sqHead = (*uint32)(unsafe.Pointer(&sqRing[params.sqOff.head]))
	sqTail = (*uint32)(unsafe.Pointer(&sqRing[params.sqOff.tail]))
	sqRingMask = (*uint32)(unsafe.Pointer(&sqRing[params.sqOff.ringMask]))
	sqRingEntries = (*uint32)(unsafe.Pointer(&sqRing[params.sqOff.ringEntries]))
	sqArray = unsafe.Slice((*uint32)(unsafe.Pointer(&sqRing[params.sqOff.array])), int(params.sqEntries))
	cqHead = (*uint32)(unsafe.Pointer(&cqRing[params.cqOff.head]))
	cqTail = (*uint32)(unsafe.Pointer(&cqRing[params.cqOff.tail]))
	cqRingMask = (*uint32)(unsafe.Pointer(&cqRing[params.cqOff.ringMask]))
	cqRingEntries = (*uint32)(unsafe.Pointer(&cqRing[params.cqOff.ringEntries]))
	cqes = unsafe.Slice((*ioUringCqe)(unsafe.Pointer(&cqRing[params.cqOff.cqes])), int(params.cqEntries))
	sqes = unsafe.Slice((*ioUringSqe)(unsafe.Pointer(&sqesMem[0])), int(params.sqEntries))
	return nil
}

func ReadFile(path string) ([]byte, error) {
	if !Enabled() {
		return os.ReadFile(path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := int(info.Size())
	if size == 0 {
		return []byte{}, nil
	}
	data := make([]byte, size)
	if err := submitRead(int(f.Fd()), data, 0); err != nil {
		return nil, err
	}
	return data, nil
}

func WriteFile(path string, data []byte, perm os.FileMode) error {
	if !Enabled() {
		return os.WriteFile(path, data, perm)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(data) == 0 {
		return nil
	}
	return submitWrite(int(f.Fd()), data, 0)
}

func submitRead(fd int, buf []byte, offset int64) error {
	mutex.Lock()
	defer mutex.Unlock()
	tail := *sqTail
	idx := tail & *sqRingMask
	sqe := &sqes[idx]
	*sqe = ioUringSqe{}
	sqe.opcode = IORING_OP_READ
	sqe.flags = 0
	sqe.ioprio = 0
	sqe.fd = int32(fd)
	sqe.off = uint64(offset)
	sqe.addr = uint64(uintptr(unsafe.Pointer(&buf[0])))
	sqe.len = uint32(len(buf))
	sqe.rwFlags = 0
	sqe.userData = uint64(idx)
	sqArray[idx] = uint32(idx)
	*sqTail = tail + 1
	if err := submitAndWait(1); err != nil {
		return err
	}
	cqe, err := popCqe()
	if err != nil {
		return err
	}
	if cqe.res < 0 {
		return syscall.Errno(-cqe.res)
	}
	return nil
}

func submitWrite(fd int, data []byte, offset int64) error {
	mutex.Lock()
	defer mutex.Unlock()
	tail := *sqTail
	idx := tail & *sqRingMask
	sqe := &sqes[idx]
	*sqe = ioUringSqe{}
	sqe.opcode = IORING_OP_WRITE
	sqe.flags = 0
	sqe.ioprio = 0
	sqe.fd = int32(fd)
	sqe.off = uint64(offset)
	sqe.addr = uint64(uintptr(unsafe.Pointer(&data[0])))
	sqe.len = uint32(len(data))
	sqe.rwFlags = 0
	sqe.userData = uint64(idx)
	sqArray[idx] = uint32(idx)
	*sqTail = tail + 1
	if err := submitAndWait(1); err != nil {
		return err
	}
	cqe, err := popCqe()
	if err != nil {
		return err
	}
	if cqe == nil {
		return os.ErrInvalid
	}
	if cqe.res < 0 {
		return syscall.Errno(-cqe.res)
	}
	return nil
}

func submitAndWait(n uint32) error {
	rc, _, err := unix.Syscall6(unix.SYS_IO_URING_ENTER, uintptr(ringFd), uintptr(n), uintptr(1), uintptr(IORING_ENTER_GETEVENTS), 0, 0)
	if int(rc) < 0 {
		return err
	}
	return nil
}

func popCqe() (*ioUringCqe, error) {
	head := *cqHead
	if head == *cqTail {
		return nil, os.ErrInvalid
	}
	cqe := &cqes[head&*cqRingMask]
	*cqHead = head + 1
	return cqe, nil
}
