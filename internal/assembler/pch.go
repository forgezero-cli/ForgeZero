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

package assembler

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
	"github.com/zeebo/blake3"
)

var PCHIncludeArgs []string
var PCHCacheDir string
var pchMutex sync.Mutex

func ResetPCH() {
	PCHIncludeArgs = nil
}

func SetPCHIncludeArgs(args []string) {
	PCHIncludeArgs = args
}

func SetPCHCacheDir(dir string) {
	PCHCacheDir = dir
}

func computePCHHash(headerPath string, compiler string, flags []string, target string) ([32]byte, error) {
	h := blake3.New()
	data, err := os.ReadFile(headerPath)
	if err != nil {
		return [32]byte{}, err
	}
	h.Write(data)
	h.Write([]byte(compiler))
	h.Write([]byte(target))
	for _, f := range flags {
		h.Write([]byte(f))
	}
	var sum [32]byte
	h.Sum(sum[:0])
	return sum, nil
}

func getPCHPath(headerPath string, compiler string, flags []string, target string) (string, error) {
	if PCHCacheDir == "" {
		return "", nil
	}
	sum, err := computePCHHash(headerPath, compiler, flags, target)
	if err != nil {
		return "", err
	}
	const hexChars = "0123456789abcdef"
	hex := make([]byte, 64)
	for i, b := range sum {
		hex[i*2] = hexChars[b>>4]
		hex[i*2+1] = hexChars[b&0x0f]
	}
	return filepath.Join(PCHCacheDir, "pch_"+string(hex)+".gch"), nil
}

func getHashPath(pchPath string) string {
	return pchPath + ".hash"
}

func savePCHHash(pchPath string, hash [32]byte) error {
	return os.WriteFile(getHashPath(pchPath), hash[:], 0644)
}

func loadPCHHash(pchPath string) ([32]byte, error) {
	var h [32]byte
	data, err := os.ReadFile(getHashPath(pchPath))
	if err != nil {
		return h, err
	}
	if len(data) != 32 {
		return h, nil
	}
	copy(h[:], data)
	return h, nil
}

func BuildPCH(ctx context.Context, headerPath, outputPath string, compiler string, flags []string, verbose bool) error {
	if err := utils.EnsureDir(outputPath); err != nil {
		return err
	}
	args := make([]string, 0, 4+len(flags))
	args = append(args, "-x", "c-header", "-o", outputPath)
	args = append(args, flags...)
	args = append(args, headerPath)
	_, err := runCommand(ctx, verbose, compiler, args...)
	return err
}

func EnsurePCH(ctx context.Context, headerPath string, compiler string, flags []string, target string, verbose bool) (string, error) {
	pchPath, err := getPCHPath(headerPath, compiler, flags, target)
	if err != nil {
		return "", err
	}
	if pchPath == "" {
		return "", nil
	}
	currentHash, err := computePCHHash(headerPath, compiler, flags, target)
	if err != nil {
		return "", err
	}
	pchMutex.Lock()
	defer pchMutex.Unlock()
	storedHash, _ := loadPCHHash(pchPath)
	if storedHash != currentHash {
		if err := BuildPCH(ctx, headerPath, pchPath, compiler, flags, verbose); err != nil {
			return "", err
		}
		if err := savePCHHash(pchPath, currentHash); err != nil {
			return "", err
		}
	}
	return pchPath, nil
}