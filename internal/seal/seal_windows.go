//go:build windows
// +build windows

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

package seal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	sealed       bool
	combinedSeal [32]byte
	allowed      sync.Map
	journalBuf   []byte
	journalPos   uint32
	stateMu      sync.RWMutex
	globalState  [32]byte
	decoy        atomic.Bool
)

var (
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")
	modadvapi32 = syscall.NewLazyDLL("advapi32.dll")

	procGetVolumeInformationW = modkernel32.NewProc("GetVolumeInformationW")
	procSetFileAttributesW    = modkernel32.NewProc("SetFileAttributesW")
)

func getVolumeSerialNumber(root string) (uint32, error) {
	rootPtr, err := syscall.UTF16PtrFromString(root + "\\")
	if err != nil {
		return 0, err
	}
	var serial uint32
	var maxCompLen uint32
	var flags uint32
	var fsName [256]uint16
	var volName [256]uint16
	ret, _, err := procGetVolumeInformationW.Call(
		uintptr(unsafe.Pointer(rootPtr)),
		uintptr(unsafe.Pointer(&volName[0])),
		uintptr(len(volName)),
		uintptr(unsafe.Pointer(&serial)),
		uintptr(unsafe.Pointer(&maxCompLen)),
		uintptr(unsafe.Pointer(&flags)),
		uintptr(unsafe.Pointer(&fsName[0])),
		uintptr(len(fsName)),
	)
	if ret == 0 {
		return 0, err
	}
	return serial, nil
}

func MachineID() (string, error) {
	root := os.Getenv("SystemDrive")
	if root == "" {
		root = "C:"
	}
	serial, err := getVolumeSerialNumber(root)
	if err != nil {
		return "", err
	}
	b := make([]byte, 4)
	b[0] = byte(serial)
	b[1] = byte(serial >> 8)
	b[2] = byte(serial >> 16)
	b[3] = byte(serial >> 24)
	return hex.EncodeToString(b), nil
}

func computeFileHash(path string) ([32]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return [32]byte{}, err
	}
	defer f.Close()
	hasher := sha256.New()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return [32]byte{}, err
	}
	var out [32]byte
	copy(out[:], hasher.Sum(nil))
	return out, nil
}

func walkProjectFiles(root string, visit func(path string, info os.FileInfo, err error) error) error {
	root = filepath.Clean(root)
	info, err := os.Lstat(root)
	if err != nil {
		return visit(root, nil, err)
	}
	if err := visit(root, info, nil); err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		path := filepath.Join(root, name)
		info, err := entry.Info()
		if err != nil {
			if err := visit(path, nil, err); err != nil {
				return err
			}
			continue
		}
		if err := visit(path, info, nil); err != nil {
			return err
		}
		if info.IsDir() {
			if err := walkProjectFiles(path, visit); err != nil {
				return err
			}
		}
	}
	return nil
}

func setImmutable(path string) error {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	ret, _, _ := procSetFileAttributesW.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(syscall.FILE_ATTRIBUTE_READONLY),
	)
	if ret == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func Seal() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	dir := filepath.Dir(execPath)
	sealPath := filepath.Join(dir, ".fz_seal")
	mid, err := MachineID()
	if err != nil {
		return err
	}
	execHash, err := computeFileHash(execPath)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	hasher.Write(execHash[:])
	hasher.Write([]byte(mid))
	sum := hasher.Sum(nil)

	file, err := os.Create(sealPath)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(hex.EncodeToString(sum) + "\n"); err != nil {
		return err
	}
	if err := walkProjectFiles(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		h, err := computeFileHash(path)
		if err != nil {
			return err
		}
		_, err = file.WriteString(hex.EncodeToString(h[:]) + "\t" + path + "\n")
		return err
	}); err != nil {
		return err
	}
	if err := setImmutable(sealPath); err != nil {
		return err
	}
	return nil
}

func Verify() (bool, error) {
	execPath, err := os.Executable()
	if err != nil {
		return false, err
	}
	dir := filepath.Dir(execPath)
	sealPath := filepath.Join(dir, ".fz_seal")
	data, err := os.ReadFile(sealPath)
	if err != nil {
		return false, nil
	}
	lines := bytes.Split(data, []byte{'\n'})
	if len(lines) == 0 {
		return false, nil
	}
	first := bytes.TrimSpace(lines[0])
	sealHash, err := hex.DecodeString(string(first))
	if err != nil || len(sealHash) != 32 {
		return false, nil
	}
	mid, err := MachineID()
	if err != nil {
		return false, err
	}
	execHash, err := computeFileHash(execPath)
	if err != nil {
		return false, err
	}
	hasher := sha256.New()
	hasher.Write(execHash[:])
	hasher.Write([]byte(mid))
	sum := hasher.Sum(nil)
	if !bytes.Equal(sealHash, sum) {
		return false, nil
	}
	stateMu.Lock()
	copy(combinedSeal[:], sealHash)
	sealed = true
	stateMu.Unlock()
	allowed = sync.Map{}
	for i := 1; i < len(lines); i++ {
		ln := bytes.TrimSpace(lines[i])
		if len(ln) == 0 {
			continue
		}
		parts := bytes.SplitN(ln, []byte{'\t'}, 2)
		if len(parts) >= 1 {
			allowed.Store(string(parts[0]), struct{}{})
		}
	}
	return true, nil
}

func getExecPath() (string, error) {
	return os.Executable()
}

func readLink(path string) (string, error) {
	return os.Readlink(path)
}

func secureMmap(size int) ([]byte, error) {
	return make([]byte, size), nil
}

func munmap([]byte) error { return nil }

func resetSealState() {
	atomic.StoreUint32(&journalPos, 0)
	if len(journalBuf) > 0 {
		for i := range journalBuf {
			journalBuf[i] = 0
		}
	}
	stateMu.Lock()
	sealed = false
	for i := range globalState {
		globalState[i] = 0
	}
	for i := range combinedSeal {
		combinedSeal[i] = 0
	}
	stateMu.Unlock()
	allowed = sync.Map{}
	decoy.Store(false)
}

func UpdateGlobalState(data []byte) {
	stateMu.Lock()
	hasher := sha256.New()
	hasher.Write(globalState[:])
	hasher.Write(data)
	tmp := hasher.Sum(nil)
	copy(globalState[:], tmp)
	stateMu.Unlock()
}

func JournalEvent(data []byte) {
	if len(journalBuf) == 0 {
		return
	}
	n := uint32(len(data))
	if n == 0 {
		return
	}
	bufLen := uint32(len(journalBuf))
	if bufLen == 0 {
		return
	}
	start := atomic.AddUint32(&journalPos, n) - n
	idx := start % bufLen
	first := int(bufLen - idx)
	if first >= int(n) {
		copy(journalBuf[idx:idx+n], data)
	} else {
		copy(journalBuf[idx:], data[:first])
		copy(journalBuf[:], data[first:])
	}
	UpdateGlobalState(data)
}

func IsDecoyMode() bool {
	return decoy.Load()
}

func IsAllowedHex(h string) bool {
	_, ok := allowed.Load(h)
	return ok
}

func SetImmutablePath(string) error {
	return nil
}

func WalkProjectFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func ComputeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hasher := sha256.New()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
