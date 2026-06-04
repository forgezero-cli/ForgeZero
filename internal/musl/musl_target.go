package musl

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// TODO:
// PLATFORM RISC-V X86_64
// ARM / AARCH64
// RISC-V [X]

//go:embed assets/musl/*
var muslAssets embed.FS

type Toolchain struct {
	TargetArch string
	tmpDir     string
}

var (
	staticFlag   = "-static"
	nostdlibFlag = "-nostdlib"
	lFlag        = "-L"
	lcFlag       = "-lc"
	oFlag        = "-o"
)

func GetLinkerArgsZeroAlloc(dst []string, muslDir string, objFiles []string, outputFile string) []string {
	args := []string{
		"-static",
		"-nostdlib",
		filepath.Join(muslDir, "crt1.o"),
		filepath.Join(muslDir, "crti.o"),
	}

	args = append(args, objFiles...)
	args = append(args,
		"-L"+muslDir,
		"-lc",
		filepath.Join(muslDir, "libgcc.a"),
		filepath.Join(muslDir, "crtn.o"),
		"-o", outputFile,
	)

	for i, arg := range args {
		if i < len(dst) {
			dst[i] = arg
		}
	}

	return dst

}

func NewToolchain(arch string) *Toolchain {
	return &Toolchain{TargetArch: arch}
}

func (t *Toolchain) Prepare() (string, error) {
	tmpDir, err := os.MkdirTemp("", "fz-musl-*")
	if err != nil {
		return "", fmt.Errorf("failed to create build temp dir: %w", err)
	}
	t.tmpDir = tmpDir

	subDir := filepath.Join("assets", "musl", t.TargetArch)

	entries, err := fs.ReadDir(muslAssets, subDir)
	if err != nil {
		t.Close()
		return "", fmt.Errorf("architecture %s is not supported by ForgeZero musl toolchain", t.TargetArch)
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
		return nil, fmt.Errorf("toolchain is not prepared, call Prepare() first")
	}

	args := []string{
		"-static",
		"-nostdlib",
		filepath.Join(t.tmpDir, "crt1.o"),
		filepath.Join(t.tmpDir, "crti.o"),
	}

	args = append(args, userObjFiles...)

	args = append(args,
		"-L"+t.tmpDir,
		"-lc",
		filepath.Join(t.tmpDir, "crtn.o"),
		"-o", outputFile,
	)

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
