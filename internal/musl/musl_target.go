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

package musl

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

//go:embed assets/musl/*
var muslAssets embed.FS

type Toolchain struct {
	TargetArch string
	tmpDir     string
}

func GetLinkerArgsZeroAlloc(dst []string, muslDir string, objFiles []string, outputFile string) []string {
	i := 0
	dst[i] = "-static"
	i++
	dst[i] = "-nostdlib"
	i++
	dst[i] = filepath.Join(muslDir, "crt1.o")
	i++
	dst[i] = filepath.Join(muslDir, "crti.o")
	i++
	for _, obj := range objFiles {
		dst[i] = obj
		i++
	}
	dst[i] = "-L" + muslDir
	i++
	dst[i] = "-lc"
	i++
	dst[i] = filepath.Join(muslDir, "libgcc.a")
	i++
	dst[i] = filepath.Join(muslDir, "crtn.o")
	i++
	dst[i] = "-o"
	i++
	dst[i] = outputFile
	i++
	return dst[:i]
}

func NewToolchain(arch string) *Toolchain {
	return &Toolchain{TargetArch: arch}
}

func (t *Toolchain) Prepare() (string, error) {
	tmpDir, err := os.MkdirTemp("", "fz-musl-*")
	if err != nil {
		return "", errors.New("failed to create build temp dir: " + err.Error())
	}
	t.tmpDir = tmpDir

	subDir := path.Join("assets", "musl", t.TargetArch)

	entries, err := fs.ReadDir(muslAssets, subDir)
	if err != nil {
		_ = t.Close()
		return "", errors.New("architecture " + t.TargetArch + " is not supported by ForgeZero musl toolchain")
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := muslAssets.ReadFile(path.Join(subDir, entry.Name()))
		if err != nil {
			_ = t.Close()
			return "", err
		}

		destPath := filepath.Join(tmpDir, entry.Name())
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			_ = t.Close()
			return "", err
		}
	}

	return tmpDir, nil
}

func (t *Toolchain) GetLinkerArgs(userObjFiles []string, outputFile string) ([]string, error) {
	if t.tmpDir == "" {
		return nil, errors.New("toolchain is not prepared, call Prepare() first")
	}

	args := make([]string, 0, len(userObjFiles)+9)
	args = append(args, "-static", "-nostdlib")
	args = append(args, filepath.Join(t.tmpDir, "crt1.o"))
	args = append(args, filepath.Join(t.tmpDir, "crti.o"))
	args = append(args, userObjFiles...)
	args = append(args, "-L"+t.tmpDir, "-lc")
	args = append(args, filepath.Join(t.tmpDir, "crtn.o"))
	args = append(args, "-o", outputFile)

	return args, nil
}

func (t *Toolchain) Close() error {
	if t.tmpDir != "" {
		err := os.RemoveAll(t.tmpDir)
		t.tmpDir = ""
		return err
	}
	return nil
}
