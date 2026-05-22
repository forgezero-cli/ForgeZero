package integration_test

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fz/internal/builder"
	"fz/internal/config"
	"fz/internal/utils"
)

func writeDummySource(dir string) (string, error) {
	p := filepath.Join(dir, "main.asm")
	data := []byte("; dummy")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return "", err
	}
	return p, nil
}

func TestEnterpriseIsolation(t *testing.T) {
	dir := t.TempDir()
	_, err := writeDummySource(dir)
	if err != nil {
		t.Fatalf("write source: %v", err)
	}

	cfg := &config.Config{}
	cfg.Isolation = config.IsolationStandard
	cfg.DeterministicStrip = true
	cfg.ToolchainSettings.SearchPriority = []string{"local", "system"}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ctx = utils.ContextWithConfig(ctx, cfg)

	toolBin := filepath.Join(dir, "toolchain", "bin")
	if err := os.MkdirAll(toolBin, 0o755); err != nil {
		t.Fatalf("mkdir toolchain: %v", err)
	}
	nasmPath := filepath.Join(toolBin, "nasm")
	ldPath := filepath.Join(toolBin, "ld")
	if err := os.WriteFile(nasmPath, []byte("#!/bin/sh\n# write OBJ_CONTENT to -o arg\nprev=\"\"\nfor arg in \"$@\"; do\n  if [ \"$prev\" = \"-o\" ]; then\n    echo -n OBJ_CONTENT > \"$arg\"\n  fi\n  prev=\"$arg\"\ndone\n"), 0o755); err != nil {
		t.Fatalf("write nasm stub: %v", err)
	}
	if err := os.WriteFile(ldPath, []byte("#!/bin/sh\n# write BIN_CONTENT to -o arg\nprev=\"\"\nfor arg in \"$@\"; do\n  if [ \"$prev\" = \"-o\" ]; then\n    echo -n BIN_CONTENT > \"$arg\"\n  fi\n  prev=\"$arg\"\ndone\n"), 0o755); err != nil {
		t.Fatalf("write ld stub: %v", err)
	}
	utils.SetExecutionRoot(dir)
	os.Setenv("FZ_TEST_VARIANT", "A")
	res1, err := builder.BuildDir(ctx, []string{dir}, filepath.Join(dir, "out1"), false, false, "raw", false, true, false, true, false, nil, nil, nil, nil, nil, 1, "executable")
	if err != nil {
		t.Fatalf("build1 failed: %v", err)
	}
	b1, err := os.ReadFile(res1.Binary)
	if err != nil {
		t.Fatalf("read bin1: %v", err)
	}
	h1 := sha256.Sum256(b1)

	os.Setenv("FZ_TEST_VARIANT", "B")
	res2, err := builder.BuildDir(ctx, []string{dir}, filepath.Join(dir, "out2"), false, false, "raw", false, true, false, true, false, nil, nil, nil, nil, nil, 1, "executable")
	if err != nil {
		t.Fatalf("build2 failed: %v", err)
	}
	b2, err := os.ReadFile(res2.Binary)
	if err != nil {
		t.Fatalf("read bin2: %v", err)
	}
	h2 := sha256.Sum256(b2)

	if h1 != h2 {
		t.Fatalf("binaries differ under isolation: %x vs %x", h1, h2)
	}
}
