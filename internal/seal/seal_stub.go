//go:build !linux && !windows
// +build !linux,!windows

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
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

func MachineID() (string, error) {
	return "", errors.New("machine ID not supported on this platform")
}

func Seal() error {
	return errors.New("seal not supported on this platform")
}

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

func getExecPath() (string, error) {
	return os.Executable()
}

func readLink(path string) (string, error) {
	return os.Readlink(path)
}

func secureMmap(size int) ([]byte, error) {
	buf := make([]byte, size)
	_, err := rand.Read(buf)
	return buf, err
}

func munmap([]byte) error { return nil }

func setImmutable(string) error { return nil }

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

func UpdateGlobalState(data []byte) {
	stateMu.Lock()
	hasher := sha256.New()
	hasher.Write(globalState[:])
	hasher.Write(data)
	tmp := hasher.Sum(nil)
	copy(globalState[:], tmp)
	stateMu.Unlock()
}

func Verify() (bool, error) {
	return false, errors.New("seal not implemented on this platform")
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
