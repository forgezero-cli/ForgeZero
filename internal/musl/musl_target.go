package musl

import (
	"embed"
	"fmt"
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

var (
	staticFlag   = "-static"
	nostdlibFlag = "-nostdlib"
	lFlag        = "-L"
	lcFlag       = "-lc"
	oFlag        = "-o"
)

func GetLinkerArgsZeroAlloc(dst []string, muslDir string, objFiles []string, outputFile string) []string {
	dst[0] = staticFlag
	dst[1] = nostdlibFlag
	dst[2] = muslDir + "/crt1.o"
	dst[3] = muslDir + "/crti.o"

	offset := 4
	for i := 0; i < len(objFiles); i++ {
		dst[offset+i] = objFiles[i]
	}

	offset += len(objFiles)
	dst[offset] = lFlag + muslDir
	dst[offset+1] = lcFlag
	dst[offset+2] = muslDir + "/crtn.o"
	dst[offset+3] = oFlag
	dst[offset+4] = outputFile

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
