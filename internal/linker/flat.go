//go:build linux
// +build linux

package linker

import (
	"context"
	"debug/elf"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"fz/internal/assembler"
	"fz/internal/utils"
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
	out, err := EmitFlatBinary(layout)
	if err != nil {
		return err
	}
	return os.WriteFile(bin, out, 0o755)
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
		return 0, fmt.Errorf("section %s is not allocatable", name)
	}
	if section.Size > uint64(^uint32(0)) {
		return 0, fmt.Errorf("section %s too large", name)
	}
	return uint32(section.Size), nil
}

func verifyNoRelocations(file *elf.File) error {
	for _, section := range file.Sections {
		switch section.Type {
		case elf.SHT_REL, elf.SHT_RELA:
			return fmt.Errorf("unsupported relocations in baremetal object: %s", section.Name)
		}
	}
	return nil
}

func copyFileHot(src, dst string) error {
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
	var buf [65536]byte
	for {
		rn, rerr := readHot(sfd, buf[:])
		if rn > 0 {
			written := 0
			for written < rn {
				wn, werr := writeHot(dfd, buf[written:rn])
				if werr != nil {
					_ = closeHot(dfd)
					_ = closeHot(sfd)
					return werr
				}
				written += wn
			}
		}
		if rerr != nil {
			if rerr == syscall.EINTR {
				continue
			}
			_ = closeHot(dfd)
			_ = closeHot(sfd)
			return rerr
		}
		if rn == 0 {
			break
		}
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
