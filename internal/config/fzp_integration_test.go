package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFZPGeneratesConfigHeaderForCCompilation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.fz")
	if err := os.WriteFile(cfgPath, []byte("#define OUTPUT app\n#define MODE raw\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFZP(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	templatePath := filepath.Join(dir, "config.h.in")
	outputPath := filepath.Join(dir, "config.h")
	if err := os.WriteFile(templatePath, []byte("#define FZ_OUTPUT \"${OUTPUT}\"\n#define FZ_MODE \"${MODE}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := GenerateConfigH(templatePath, outputPath, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatal(err)
	}
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for filepath.Base(repoRoot) != "ForgeZero" {
		repoRoot = filepath.Dir(repoRoot)
		if repoRoot == "/" || repoRoot == "." {
			break
		}
	}
	testSource := filepath.Join(repoRoot, "testdata", "fzp", "test.c")
	cc := exec.Command("cc", "-I", dir, testSource, "-o", filepath.Join(dir, "testbin"))
	cc.Dir = dir
	if out, err := cc.CombinedOutput(); err != nil {
		t.Fatalf("compile failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "testbin")); err != nil {
		t.Fatal(err)
	}
}
