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
	"sync"
	"syscall"
	"unsafe"

	"github.com/forgezero-cli/ForgeZero/internal/logger"
	"golang.org/x/sys/unix"
)

const (
	ioUringSetupEntries       = 256
	IORING_ENTER_GETEVENTS    = 1
	IORING_OFF_SQ_RING        = 0
	IORING_OFF_CQ_RING        = 0x8000000
	IORING_OFF_SQES           = 0x10000000
	IORING_OP_READ            = 22
	IORING_OP_WRITE           = 23
	IORING_OP_STATX           = 21
	IORING_SETUP_COOP_TASKRUN = 1 << 8
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
	params := ioUringParams{sqEntries: ioUringSetupEntries, cqEntries: ioUringSetupEntries, flags: IORING_SETUP_COOP_TASKRUN}
	fd, _, errno := unix.Syscall(unix.SYS_IO_URING_SETUP, uintptr(ioUringSetupEntries), uintptr(unsafe.Pointer(&params)), 0)
	if int(fd) < 0 && errno == unix.EINVAL {
		params = ioUringParams{sqEntries: ioUringSetupEntries, cqEntries: ioUringSetupEntries}
		fd, _, errno = unix.Syscall(unix.SYS_IO_URING_SETUP, uintptr(ioUringSetupEntries), uintptr(unsafe.Pointer(&params)), 0)
	}
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
		return os.ReadFile(path)
	}
	return data, nil
}

func ReadFiles(paths []string) ([][]byte, error) {
	if !Enabled() {
		return readFilesFallback(paths)
	}
	if len(paths) == 0 {
		return [][]byte{}, nil
	}
	if len(paths) > int(*sqRingEntries) {
		return readFilesFallback(paths)
	}
	files := make([]*os.File, len(paths))
	data := make([][]byte, len(paths))
	for i, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			for _, opened := range files {
				if opened != nil {
					_ = opened.Close()
				}
			}
			return nil, err
		}
		files[i] = file
		info, err := file.Stat()
		if err != nil {
			for _, opened := range files {
				if opened != nil {
					_ = opened.Close()
				}
			}
			return nil, err
		}
		if info.Size() > int64(^uint(0)>>1) {
			for _, opened := range files {
				if opened != nil {
					_ = opened.Close()
				}
			}
			return nil, os.ErrInvalid
		}
		if uint64(info.Size()) > uint64(^uint32(0)) {
			for _, opened := range files {
				if opened != nil {
					_ = opened.Close()
				}
			}
			return readFilesFallback(paths)
		}
		data[i] = make([]byte, int(info.Size()))
	}
	mutex.Lock()
	tail := *sqTail
	queued := uint32(0)
	for i, file := range files {
		if len(data[i]) == 0 {
			continue
		}
		idx := tail + queued
		sqe := &sqes[idx&*sqRingMask]
		*sqe = ioUringSqe{}
		sqe.opcode = IORING_OP_READ
		sqe.fd = int32(file.Fd())
		sqe.addr = uint64(uintptr(unsafe.Pointer(&data[i][0])))
		sqe.len = uint32(len(data[i]))
		sqe.userData = uint64(i)
		sqArray[idx&*sqRingMask] = idx & *sqRingMask
		queued++
	}
	*sqTail = tail + queued
	if queued == 0 {
		mutex.Unlock()
		for _, file := range files {
			_ = file.Close()
		}
		return data, nil
	}
	err := submitBatchAndWait(queued)
	if err == nil {
		for range queued {
			cqe, popErr := popCqe()
			if popErr != nil {
				if err == nil {
					err = popErr
				}
				continue
			}
			index := int(cqe.userData)
			if index < 0 || index >= len(data) {
				if err == nil {
					err = os.ErrInvalid
				}
				continue
			}
			if cqe.res < 0 {
				if err == nil {
					err = syscall.Errno(-cqe.res)
				}
				continue
			}
			if int(cqe.res) != len(data[index]) {
				if err == nil {
					err = io.ErrUnexpectedEOF
				}
			}
		}
	}
	mutex.Unlock()
	for _, file := range files {
		_ = file.Close()
	}
	if err != nil {
		return readFilesFallback(paths)
	}
	return data, nil
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
	if err := submitWrite(int(f.Fd()), data, 0); err != nil {
		return os.WriteFile(path, data, perm)
	}
	return nil
}

func StatxAt(dirFD int, path string, flags int, mask int) (StatxResult, error) {
	var statx unix.Statx_t
	result := StatxResult{}
	if !Enabled() {
		err := unix.Statx(dirFD, path, flags, mask, &statx)
		if err == nil {
			result = convertStatx(statx)
		}
		return result, err
	}
	mutex.Lock()
	tail := *sqTail
	index := tail & *sqRingMask
	sqe := &sqes[index]
	*sqe = ioUringSqe{}
	sqe.opcode = IORING_OP_STATX
	sqe.fd = int32(dirFD)
	sqe.addr = uint64(uintptr(unsafe.Pointer(unsafe.StringData(path))))
	sqe.off = uint64(uintptr(unsafe.Pointer(&statx)))
	sqe.len = uint32(mask)
	sqe.rwFlags = uint32(flags)
	sqe.userData = uint64(index)
	sqArray[index] = index
	*sqTail = tail + 1
	err := submitAndWait(1)
	if err == nil {
		var cqe *ioUringCqe
		cqe, err = popCqe()
		if err == nil && cqe.res < 0 {
			err = syscall.Errno(-cqe.res)
		}
		if err == nil && statx.Mask == 0 {
			err = os.ErrInvalid
		}
	}
	mutex.Unlock()
	if err != nil {
		err = unix.Statx(dirFD, path, flags, mask, &statx)
		if err == nil {
			result = convertStatx(statx)
		}
		return result, err
	}
	result = convertStatx(statx)
	return result, nil
}

func convertStatx(statx unix.Statx_t) StatxResult {
	return StatxResult{Mask: statx.Mask, Mode: statx.Mode, Ino: statx.Ino, Size: statx.Size, MtimeSec: statx.Mtime.Sec, MtimeNsec: statx.Mtime.Nsec}
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
	return validateResult(cqe, len(buf), io.ErrUnexpectedEOF)
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
	return validateResult(cqe, len(data), io.ErrShortWrite)
}

func validateResult(cqe *ioUringCqe, expected int, shortErr error) error {
	if cqe == nil {
		return os.ErrInvalid
	}
	if cqe.res < 0 {
		return syscall.Errno(-cqe.res)
	}
	if int(cqe.res) != expected {
		return shortErr
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

func submitBatchAndWait(n uint32) error {
	rc, _, err := unix.Syscall6(unix.SYS_IO_URING_ENTER, uintptr(ringFd), uintptr(n), uintptr(n), uintptr(IORING_ENTER_GETEVENTS), 0, 0)
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
