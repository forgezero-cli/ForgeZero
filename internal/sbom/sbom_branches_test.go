package sbom

import (
	"os"
	"path/filepath"
	"testing"

	"fz/internal/config"
	"fz/internal/utils"
)

func TestGenerateEmptyRootUsesCwd(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	sb, err := Generate("", "vendor", "1.0", nil, "x86_64-linux-gnu")
	if err != nil {
		t.Fatal(err)
	}
	if sb == nil {
		t.Fatal("nil sbom")
	}
}

func TestScanVendorMissing(t *testing.T) {
	root := t.TempDir()
	components, err := scanVendorComponents(root, "vendor")
	if err != nil {
		t.Fatal(err)
	}
	if components != nil {
		t.Fatalf("expected nil, got %v", components)
	}
}

func TestScanVendorNotDirectory(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor")
	if err := os.WriteFile(vendor, []byte("x"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	_, err := scanVendorComponents(root, "vendor")
	if err == nil {
		t.Fatal("expected not directory error")
	}
}

func TestHashVendorEntryFile(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor")
	if err := os.MkdirAll(vendor, utils.DirPerm); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(vendor, "lib.txt")
	if err := os.WriteFile(file, []byte("pkg"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	entry, err := os.ReadDir(vendor)
	if err != nil {
		t.Fatal(err)
	}
	h, err := hashVendorEntry(root, file, entry[0])
	if err != nil {
		t.Fatal(err)
	}
	if h == "" {
		t.Fatal("empty hash")
	}
}

func TestGenerateWithToolChecksums(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{ToolChecksums: map[string]string{"gcc": "abc"}}
	sb, err := Generate(root, "vendor", "3.1", cfg, "wasm32-wasi")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range sb.Metadata.Properties {
		if p.Name == "tool.checksum.gcc" {
			found = true
		}
	}
	if !found {
		t.Fatal("checksum property missing")
	}
}

func TestMarshalNilSafe(t *testing.T) {
	sb := &SBOM{BomFormat: "CycloneDX", SpecVersion: "1.4", Version: 1}
	data, err := Marshal(sb)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty")
	}
}
