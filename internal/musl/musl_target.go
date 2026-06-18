package musl

import (
	"embed"
	"errors"
	"io/fs"
	"os"
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

	subDir := filepath.Join("assets", "musl", t.TargetArch)

	entries, err := fs.ReadDir(muslAssets, subDir)
	if err != nil {
		t.Close()
		return "", errors.New("architecture " + t.TargetArch + " is not supported by ForgeZero musl toolchain")
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := muslAssets.ReadFile(filepath.Join(subDir, entry.Name()))
		if err != nil {
			t.Close()
			return "", err
		}

		destPath := filepath.Join(tmpDir, entry.Name())
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			t.Close()
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