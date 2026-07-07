//go:build linux
// +build linux

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
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/zeebo/blake3"
	"golang.org/x/sys/unix"
)


var sealed bool
var combinedSeal [32]byte
var allowed sync.Map // map[string]struct{}
var journalBuf []byte
var journalPos uint32
var stateMu sync.RWMutex
var globalState [32]byte
var decoy atomic.Bool
var machineIDPath = "/etc/machine-id"


func init() {
	buf, err := syscall.Mmap(-1, 0, 1<<20, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err == nil {
		journalBuf = buf
	} else {
		journalBuf = make([]byte, 1<<20)
	}
}

func getExecPath() (string, error) {
	var buf [4096]byte
	n, err := unix.Readlink("/proc/self/exe", buf[:])
	if err != nil {
		return "", err
	}
	return filepath.Clean(string(buf[:n])), nil
}

func getMachineIDZeroAlloc() (string, error) {
	fd, err := unix.Open(machineIDPath, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return "", err
	}
	defer unix.Close(fd)
	var buf [128]byte
	n, err := unix.Read(fd, buf[:])
	if err != nil && err != unix.EINTR && n == 0 {
		return "", err
	}
	return string(bytes.TrimSpace(buf[:n])), nil
}

func computeFileHash(path string) ([32]byte, error) {
	var out [32]byte
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return out, err
	}
	defer unix.Close(fd)
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return out, err
	}
	hasher := blake3.New()
	if st.Size > 0 {
		data, err := syscall.Mmap(int(fd), 0, int(st.Size), syscall.PROT_READ, syscall.MAP_PRIVATE)
		if err == nil {
			if _, err := hasher.Write(data); err != nil {
				_ = syscall.Munmap(data)
				return out, err
			}
			if err := syscall.Munmap(data); err != nil {
				return out, err
			}
		} else {
			var buf [32768]byte
			for {
				n, err := unix.Read(fd, buf[:])
				if n > 0 {
					if _, err := hasher.Write(buf[:n]); err != nil {
						return out, err
					}
				}
				if err != nil {
					if err == unix.EINTR {
						continue
					}
					if err == io.EOF {
						break
					}
					return out, err
				}
			}
		}
	}
	sum := hasher.Sum(nil)
	copy(out[:], sum[:32])
	return out, nil
}

func writeAll(fd int, b []byte) error {
	for len(b) > 0 {
		n, err := unix.Write(fd, b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

func setImmutable(path string) error {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	const FS_IMMUTABLE_FL = 0x00000010
	return unix.IoctlSetInt(fd, unix.FS_IOC_SETFLAGS, FS_IMMUTABLE_FL)
}

func isStagingMode() bool {
	return os.Getenv("FZ_STAGING") == "1"
}

func MachineID() (string, error) {
	return getMachineIDZeroAlloc()
}

func debuggerPresent() bool {
	if os.Getenv("FZ_DEBUGGER_SIMULATE") == "1" {
		return true
	}
	fd, err := unix.Open("/proc/self/status", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return false
	}
	defer unix.Close(fd)
	var buf [512]byte
	n, err := unix.Read(fd, buf[:])
	if n <= 0 {
		return false
	}
	if err != nil && err != unix.EINTR {
		return false
	}
	data := buf[:n]
	start := 0
	for i := 0; i <= n; i++ {
		if i == n || data[i] == '\n' {
			line := data[start:i]
			if len(line) >= 10 && bytes.HasPrefix(line, []byte("TracerPid:")) {
				val := bytes.TrimSpace(line[10:])
				return !bytes.Equal(val, []byte("0"))
			}
			start = i + 1
		}
	}
	return false
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

func zeroizeRegion(data []byte) {
	if len(data) == 0 {
		return
	}
	ptr := unsafe.Pointer(&data[0])
	for i := 0; i < len(data); i++ {
		*(*byte)(unsafe.Add(ptr, uintptr(i))) = 0
	}
}

func triggerDecoy() {
	if decoy.Load() {
		return
	}
	decoy.Store(true)
	atomic.StoreUint32(&journalPos, 0)
	if len(journalBuf) > 0 {
		zeroizeRegion(journalBuf)
	}
	stateMu.Lock()
	zeroizeRegion(globalState[:])
	zeroizeRegion(combinedSeal[:])
	stateMu.Unlock()
}

func resetSealState() {
	atomic.StoreUint32(&journalPos, 0)
	if len(journalBuf) > 0 {
		zeroizeRegion(journalBuf)
	}
	stateMu.Lock()
	sealed = false
	zeroizeRegion(globalState[:])
	zeroizeRegion(combinedSeal[:])
	stateMu.Unlock()
	allowed = sync.Map{}
	decoy.Store(false)
}

func UpdateGlobalState(data []byte) {
	stateMu.Lock()
	hasher := blake3.New()
	if _, err := hasher.Write(globalState[:]); err != nil {
		stateMu.Unlock()
		return
	}
	if _, err := hasher.Write(data); err != nil {
		stateMu.Unlock()
		return
	}
	sum := hasher.Sum(nil)
	copy(globalState[:], sum[:32])
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
	start := atomic.AddUint32(&journalPos, n) - n
	if bufLen == 0 {
		return
	}
	idx := start % bufLen
	first := int(bufLen - idx)
	if first >= int(n) {
		copy(journalBuf[idx:int(idx)+int(n)], data)
	} else {
		copy(journalBuf[idx:], data[:first])
		copy(journalBuf[0:], data[first:])
	}
	UpdateGlobalState(data)
}

func Seal() error {
	if debuggerPresent() && !isStagingMode() {
		triggerDecoy()
		return nil
	}
	execPath, err := getExecPath()
	if err != nil {
		return err
	}
	execHash, err := computeFileHash(execPath)
	if err != nil {
		return err
	}
	mid, err := getMachineIDZeroAlloc()
	if err != nil {
		return err
	}
	hasher := blake3.New()
	if _, err := hasher.Write(execHash[:]); err != nil {
		return err
	}
	if _, err := hasher.Write([]byte(mid)); err != nil {
		return err
	}
	sum := hasher.Sum(nil)
	stateMu.Lock()
	copy(combinedSeal[:], sum[:32])
	stateMu.Unlock()
	dir := filepath.Dir(execPath)
	sealPath := filepath.Join(dir, ".fz_seal")
	fd, err := unix.Open(sealPath, unix.O_WRONLY|unix.O_CREAT|unix.O_TRUNC|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	hexBuf := make([]byte, hex.EncodedLen(32))
	hex.Encode(hexBuf, sum[:32])
	if err := writeAll(fd, hexBuf); err != nil {
		return err
	}
	if err := writeAll(fd, []byte{'\n'}); err != nil {
		return err
	}

	root := filepath.Dir(execPath)
	if err := walkProjectFiles(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		h, err := computeFileHash(path)
		if err != nil {
			return err
		}
		JournalEvent(h[:])
		hexBuf := make([]byte, hex.EncodedLen(32))
		hex.Encode(hexBuf, h[:])
		if err := writeAll(fd, hexBuf); err != nil {
			return err
		}
		if err := writeAll(fd, []byte{'\t'}); err != nil {
			return err
		}
		if err := writeAll(fd, []byte(path)); err != nil {
			return err
		}
		if err := writeAll(fd, []byte{'\n'}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	if !isStagingMode() {
		if err := setImmutable(sealPath); err != nil {
			return err
		}
	}
	stateMu.Lock()
	sealed = true
	stateMu.Unlock()
	return nil
}

func Verify() (bool, error) {
	stateMu.RLock()
	if sealed {
		stateMu.RUnlock()
		return true, nil
	}
	stateMu.RUnlock()
	if debuggerPresent() && !isStagingMode() {
		triggerDecoy()
		return true, nil
	}
	execPath, err := getExecPath()
	if err != nil {
		return false, err
	}
	dir := filepath.Dir(execPath)
	sealPath := filepath.Join(dir, ".fz_seal")
	fd, err := unix.Open(sealPath, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return false, err
	}
	defer unix.Close(fd)
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return false, err
	}
	size := int(st.Size)
	if size <= 0 {
		return false, nil
	}
	buf := make([]byte, size)
	off := 0
	for off < size {
		n, err := unix.Read(fd, buf[off:])
		if n > 0 {
			off += n
		}
		if err != nil {
			break
		}
	}
	lines := bytes.Split(buf[:off], []byte{'\n'})
	if len(lines) == 0 {
		return false, nil
	}
	first := bytes.TrimSpace(lines[0])
	sealHash := make([]byte, 32)
	if _, err := hex.Decode(sealHash, first); err != nil || len(sealHash) != 32 {
		return false, nil
	}
	execHash, err := computeFileHash(execPath)
	if err != nil {
		return false, err
	}
	mid, err := getMachineIDZeroAlloc()
	if err != nil {
		return false, err
	}
	hasher := blake3.New()
	if _, err := hasher.Write(execHash[:]); err != nil {
		return false, err
	}
	if _, err := hasher.Write([]byte(mid)); err != nil {
		return false, err
	}
	sum := hasher.Sum(nil)
	var local [32]byte
	copy(local[:], sum[:32])
	if !bytes.Equal(local[:], sealHash) {
		if debuggerPresent() && !isStagingMode() {
			triggerDecoy()
			return true, nil
		}
		return false, nil
	}
	stateMu.Lock()
	copy(combinedSeal[:], local[:])
	sealed = true
	stateMu.Unlock()
	allowed = sync.Map{}
	for i := 1; i < len(lines); i++ {
		lnB := bytes.TrimSpace(lines[i])
		if len(lnB) == 0 {
			continue
		}
		parts := bytes.SplitN(lnB, []byte{'\t'}, 2)
		if len(parts) >= 1 {
			allowed.Store(string(parts[0]), struct{}{})
		}
	}
	return true, nil
}

func GetCombined() [32]byte {
	stateMu.RLock()
	out := combinedSeal
	stateMu.RUnlock()
	return out
}

func getGlobalState() [32]byte {
	stateMu.RLock()
	out := globalState
	stateMu.RUnlock()
	return out
}

func IsDecoyMode() bool {
	return decoy.Load()
}

func IsAllowedHex(h string) bool {
	_, ok := allowed.Load(h)
	return ok
}
