//go:build linux
// +build linux

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

package linker

import (
	"context"
	"debug/elf"
	"errors"
	"os"
	"syscall"
	"unsafe"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
	"github.com/forgezero-cli/ForgeZero/internal/forge"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func linkFlatBinary(ctx context.Context, obj, bin string) error {
	if obj == bin {
		return nil
	}
	if err := utils.EnsureDir(bin); err != nil {
		return err
	}
	if IsBareMetalTarget() {
		return linkBaremetalBinary(ctx, obj, bin)
	}
	return copyFileHot(obj, bin)
}

func linkBaremetalBinary(ctx context.Context, obj, bin string) error {
	profile, ok := TargetProfileFor(Target)
	if !ok {
		return copyFileHot(obj, bin)
	}
	f, err := os.Open(obj)
	if err != nil {
		return err
	}
	defer f.Close()
	elfFile, err := elf.NewFile(f)
	if err != nil {
		return copyFileHot(obj, bin)
	}
	if err := verifyNoRelocations(elfFile); err != nil {
		return err
	}
	textData, err := loadELFSection(elfFile, ".text")
	if err != nil {
		return err
	}
	dataData, err := loadELFSection(elfFile, ".data")
	if err != nil {
		return err
	}
	bssSize, err := bssSectionSize(elfFile, ".bss")
	if err != nil {
		return err
	}
	layout, err := NewNakedMemoryLayout(profile.Flash, profile.Ram, textData, dataData, bssSize)
	if err != nil {
		return err
	}
	return writeFlatBinaryFile(bin, layout, 0o755)
}

func writeFlatBinaryFile(path string, layout Layout, mode os.FileMode) error {
	ordered, regionSizes, totalSize, err := flatBinarySize(layout)
	if err != nil {
		return err
	}
	if uint64(totalSize) < forge.HugePageSize || uint64(totalSize) > ^uint64(0)-(forge.HugePageSize-1) {
		out := make([]byte, totalSize)
		writeFlatBinaryInto(out, layout, ordered, regionSizes)
		return os.WriteFile(path, out, mode)
	}
	mappedSize := (uint64(totalSize) + forge.HugePageSize - 1) &^ (forge.HugePageSize - 1)
	mapped := forge.MmapHugepages(mappedSize)
	if mapped == nil {
		out := make([]byte, totalSize)
		writeFlatBinaryInto(out, layout, ordered, regionSizes)
		return os.WriteFile(path, out, mode)
	}
	defer forge.MunmapHugepages(mapped, mappedSize)
	buffer := unsafe.Slice((*byte)(mapped), totalSize)
	writeFlatBinaryInto(buffer, layout, ordered, regionSizes)
	return os.WriteFile(path, buffer, mode)
}

func loadELFSection(file *elf.File, name string) ([]byte, error) {
	section := file.Section(name)
	if section == nil {
		return nil, nil
	}
	data, err := section.Data()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func bssSectionSize(file *elf.File, name string) (uint32, error) {
	section := file.Section(name)
	if section == nil {
		return 0, nil
	}
	if section.Flags&elf.SHF_ALLOC == 0 {
		return 0, errors.New("section is not allocatable")
	}
	if section.Size > uint64(^uint32(0)) {
		return 0, errors.New("section too large")
	}
	return uint32(section.Size), nil
}

func verifyNoRelocations(file *elf.File) error {
	for _, section := range file.Sections {
		switch section.Type {
		case elf.SHT_REL, elf.SHT_RELA:
			return errors.New("unsupported relocations in baremetal object")
		}
	}
	return nil
}

func copyFileHot(src, dst string) error {
	const directCopyLimit = 64 * 1024
	sfd, err := openHot(src, syscall.O_RDONLY, 0)
	if err != nil {
		return err
	}
	var st syscall.Stat_t
	if err := fstatHot(sfd, &st); err != nil {
		_ = closeHot(sfd)
		return err
	}
	mode := uint32(st.Mode & 0777)
	dfd, err := openHot(dst, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, mode)
	if err != nil {
		_ = closeHot(sfd)
		return err
	}
	if st.Size <= directCopyLimit {
		err := copyRWHot(sfd, dfd)
		closeErr := closeHot(dfd)
		if closeErr == nil {
			closeErr = closeHot(sfd)
		} else {
			_ = closeHot(sfd)
		}
		if err != nil {
			_ = unlinkHot(dst)
			return err
		}
		return closeErr
	}
	if err := copySpliceHot(sfd, dfd); err != nil {
		_ = closeHot(dfd)
		_ = closeHot(sfd)
		_ = unlinkHot(dst)
		return copyFileHotRW(src, dst, mode)
	}
	if err := closeHot(dfd); err != nil {
		_ = closeHot(sfd)
		return err
	}
	if err := closeHot(sfd); err != nil {
		return err
	}
	return nil
}

func copyFileHotRW(src, dst string, mode uint32) error {
	sfd, err := openHot(src, syscall.O_RDONLY, 0)
	if err != nil {
		return err
	}
	dfd, err := openHot(dst, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, mode)
	if err != nil {
		_ = closeHot(sfd)
		return err
	}
	err = copyRWHot(sfd, dfd)
	closeErr := closeHot(dfd)
	if closeErr == nil {
		closeErr = closeHot(sfd)
	} else {
		_ = closeHot(sfd)
	}
	if err != nil {
		return err
	}
	return closeErr
}

func copyRWHot(sfd, dfd int) error {
	var buf [65536]byte
	for {
		rn, rerr := readHot(sfd, buf[:])
		if rn > 0 {
			written := 0
			for written < rn {
				wn, werr := writeHot(dfd, buf[written:rn])
				if werr != nil {
					return werr
				}
				written += wn
			}
		}
		if rerr != nil {
			if rerr == syscall.EINTR {
				continue
			}
			return rerr
		}
		if rn == 0 {
			break
		}
	}
	return nil
}

func copySpliceHot(sfd, dfd int) error {
	var pipefd [2]int
	if err := pipe2Hot(pipefd[:], syscall.O_CLOEXEC); err != nil {
		return err
	}
	defer func() {
		_ = closeHot(pipefd[0])
		_ = closeHot(pipefd[1])
	}()
	for {
		n, err := spliceHot(sfd, pipefd[1], 1<<20)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return err
		}
		if n == 0 {
			break
		}
		written := 0
		for written < n {
			wn, err := spliceHot(pipefd[0], dfd, n-written)
			if err != nil {
				if err == syscall.EINTR {
					continue
				}
				return err
			}
			written += wn
		}
	}
	return nil
}

func pipe2Hot(fds []int, flags int) error {
	if len(fds) != 2 {
		return syscall.EINVAL
	}
	return syscall.Pipe2(fds, flags)
}

func spliceHot(rfd int, wfd int, length int) (int, error) {
	r, err := syscall.Splice(rfd, nil, wfd, nil, length, 0)
	if err != nil {
		return 0, err
	}
	return int(r), nil
}

const atFDCWD = ^uintptr(99)

func fstatHot(fd int, st *syscall.Stat_t) error {
	_, _, errno := syscall.Syscall(syscall.SYS_FSTAT, uintptr(fd), uintptr(unsafe.Pointer(st)), 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func readHot(fd int, buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	r, _, errno := syscall.Syscall(syscall.SYS_READ, uintptr(fd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if errno != 0 {
		return 0, errno
	}
	return int(r), nil
}

func writeHot(fd int, buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	w, _, errno := syscall.Syscall(syscall.SYS_WRITE, uintptr(fd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if errno != 0 {
		return 0, errno
	}
	return int(w), nil
}

func closeHot(fd int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_CLOSE, uintptr(fd), 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func openHot(path string, flags int, perm uint32) (int, error) {
	var buf [4096]byte
	n := copy(buf[:], path)
	if n >= len(buf) {
		return -1, syscall.ENAMETOOLONG
	}
	buf[n] = 0
	fd, _, errno := syscall.Syscall6(syscall.SYS_OPENAT, atFDCWD, uintptr(unsafe.Pointer(&buf[0])), uintptr(flags), uintptr(perm), 0, 0)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

func unlinkHot(path string) error {
	var buf [4096]byte
	n := copy(buf[:], path)
	if n >= len(buf) {
		return syscall.ENAMETOOLONG
	}
	buf[n] = 0
	_, _, errno := syscall.Syscall(syscall.SYS_UNLINKAT, atFDCWD, uintptr(unsafe.Pointer(&buf[0])), 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func shouldSkipLinker() bool {
	return assembler.SkipLinker()
}
