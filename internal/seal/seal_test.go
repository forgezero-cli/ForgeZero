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
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestMachineIDAvailable(t *testing.T) {
	_, err := MachineID()
	if err != nil {
		t.Skip("machine id unavailable")
	}
}

func TestDebugSimulationSealBypassesNormalOperation(t *testing.T) {
	os.Setenv("FZ_DEBUGGER_SIMULATE", "1")
	defer os.Unsetenv("FZ_DEBUGGER_SIMULATE")

	if err := Seal(); err != nil {
		t.Fatalf("expected Seal to return nil in debug simulation: %v", err)
	}
}

func TestConcurrentStateUpdates(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			UpdateGlobalState([]byte(fmt.Sprintf("/tmp/test-%d", i)))
		}(i)
	}
	wg.Wait()
}

func TestSealIntegrity(t *testing.T) {
	resetSealState()
	UpdateGlobalState([]byte("config=foo"))
	first := getGlobalState()
	resetSealState()
	UpdateGlobalState([]byte("config=fop"))
	second := getGlobalState()
	if bytes.Equal(first[:], second[:]) {
		t.Fatal("expected different state for different config data")
	}
}

func TestMachineIDBinding(t *testing.T) {
	oldMachineIDPath := machineIDPath
	defer func() { machineIDPath = oldMachineIDPath }()

	tmp := t.TempDir()
	idFile := filepath.Join(tmp, "machine-id")
	if err := os.WriteFile(idFile, []byte("machine-1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	machineIDPath = idFile
	os.Setenv("FZ_STAGING", "1")
	defer os.Unsetenv("FZ_STAGING")

	resetSealState()
	if err := Seal(); err != nil {
		t.Fatal(err)
	}
	resetSealState()
	ok, err := Verify()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected verify to succeed after sealing")
	}

	if err := os.WriteFile(idFile, []byte("machine-2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resetSealState()
	ok, err = Verify()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected verify to fail after machine-id changed")
	}
}

func TestParallelHashing(t *testing.T) {
	tmp := t.TempDir()
	const total = 1000
	files := make([]string, total)
	for i := 0; i < total; i++ {
		path := filepath.Join(tmp, fmt.Sprintf("file-%04d.txt", i))
		data := []byte(fmt.Sprintf("content-%d", i))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		files[i] = path
	}

	hashes := make([][32]byte, total)
	var wg sync.WaitGroup
	for i, path := range files {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			h, err := computeFileHash(p)
			if err != nil {
				t.Errorf("hash %d failed: %v", i, err)
				return
			}
			hashes[i] = h
		}(i, path)
	}
	wg.Wait()
	if t.Failed() {
		return
	}

	for i, path := range files {
		want, err := computeFileHash(path)
		if err != nil {
			t.Fatal(err)
		}
		if hashes[i] != want {
			t.Fatalf("hash mismatch for %s", path)
		}
	}
}

func TestTriggerDecoyAndIsDecoyMode(t *testing.T) {
	resetSealState()
	triggerDecoy()
	if !IsDecoyMode() {
		t.Fatal("expected decoy mode after triggerDecoy")
	}
}

func TestJournalEventUpdatesGlobalState(t *testing.T) {
	resetSealState()
	JournalEvent([]byte("hello"))
	if atomic.LoadUint32(&journalPos) != 5 {
		t.Fatalf("expected journalPos 5, got %d", atomic.LoadUint32(&journalPos))
	}
	state := getGlobalState()
	if bytes.Equal(state[:], make([]byte, 32)) {
		t.Fatal("expected global state to change after journal event")
	}
}

func TestWalkProjectFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "one.txt"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "two.txt"), []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}

	var paths []string
	err := walkProjectFiles(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		paths = append(paths, filepath.Base(path))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 files, got %d", len(paths))
	}
}

func TestWalkProjectFilesMissingRoot(t *testing.T) {
	err := walkProjectFiles(filepath.Join(t.TempDir(), "missing"), func(path string, info os.FileInfo, err error) error {
		if err == nil {
			t.Fatal("expected error for missing root")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestComputeFileHashEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := computeFileHash(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(h[:], make([]byte, 32)) {
		t.Fatal("expected non-zero hash for empty file")
	}
}

func TestSetImmutablePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "immutable.txt")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := setImmutable(path); err != nil {
		t.Logf("setImmutable returned expected error: %v", err)
	}
}

func TestVerifyAllowedHex(t *testing.T) {
	oldMachineIDPath := machineIDPath
	defer func() { machineIDPath = oldMachineIDPath }()

	tmp := t.TempDir()
	idFile := filepath.Join(tmp, "machine-id")
	if err := os.WriteFile(idFile, []byte("machine-x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	machineIDPath = idFile
	os.Setenv("FZ_STAGING", "1")
	defer os.Unsetenv("FZ_STAGING")

	execPath, err := getExecPath()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(execPath)
	targetFile := filepath.Join(root, "seal-test-file.txt")
	if err := os.WriteFile(targetFile, []byte("sealdata"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(targetFile)

	resetSealState()
	if err := Seal(); err != nil {
		t.Fatal(err)
	}
	resetSealState()
	ok, err := Verify()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected verify to succeed")
	}
	combined := GetCombined()
	if bytes.Equal(combined[:], make([]byte, 32)) {
		t.Fatal("expected combined seal to be set after verify")
	}

	h, err := computeFileHash(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	hexHash := hex.EncodeToString(h[:])
	if !IsAllowedHex(hexHash) {
		t.Fatalf("expected %s to be allowed", hexHash)
	}
}
