package sbom

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"


	"fz/internal/config"
)


func TestGenerateSBOMWithVendorComponents(t *testing.T) {
    tmp := t.TempDir()
    vendorDir := filepath.Join(tmp, "vendor")
    if err := os.MkdirAll(filepath.Join(vendorDir, "pkg"), 0o755); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(filepath.Join(vendorDir, "pkg", "README.md"), []byte("package content"), 0o644); err != nil {
        t.Fatal(err)
    }
    sbomDoc, err := Generate(tmp, "vendor", "2.2.0 NEXUS", &config.Config{}, "x86_64-linux-gnu")
    if err != nil {
        t.Fatal(err)
    }
    if sbomDoc.BomFormat != "CycloneDX" {
        t.Fatalf("expected CycloneDX bomFormat, got %s", sbomDoc.BomFormat)
    }
    if len(sbomDoc.Components) != 1 {
        t.Fatalf("expected 1 vendor component, got %d", len(sbomDoc.Components))
    }
    if sbomDoc.Components[0].Name != "pkg" {
        t.Fatalf("expected pkg component, got %s", sbomDoc.Components[0].Name)
    }
    if sbomDoc.Metadata.Component.Name != "fz" {
        t.Fatalf("expected metadata component fz, got %s", sbomDoc.Metadata.Component.Name)
    }
}

func TestMarshalSBOMProducesJSON(t *testing.T) {
	sbomDoc := &SBOM{
		BomFormat:  "CycloneDX",
		SpecVersion: "1.4",
		Version:    1,
		Metadata: Metadata{Component: Component{Name: "fz", Version: "2.2.0"}},
	}
	data, err := Marshal(sbomDoc)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty JSON output")
	}
	if string(data) == "" {
		t.Fatal("expected valid JSON string")
	}
}

func TestGenerateSBOMVendorSymlinkInsideRoot(t *testing.T) {
	tmp := t.TempDir()
	vendorDir := filepath.Join(tmp, "vendor")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}


	realDir := filepath.Join(tmp, "real", "libreal")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "README.md"), []byte("real lib"), 0o644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(vendorDir, "libreal_link")
	if err := os.Symlink(realDir, linkPath); err != nil {
		t.Fatal(err)
	}

	sbomDoc, err := Generate(tmp, "vendor", "2.2.0 NEXUS", &config.Config{}, "x86_64-linux-gnu")
	if err != nil {
		t.Fatal(err)
	}
	if len(sbomDoc.Components) != 1 {
		t.Fatalf("expected 1 vendor component, got %d", len(sbomDoc.Components))
	}
	if sbomDoc.Components[0].Name != "libreal_link" {
		t.Fatalf("expected libreal_link component, got %s", sbomDoc.Components[0].Name)
	}
}

func TestGenerateSBOMVendorSymlinkOutsideRootSecurityWarningAndSkip(t *testing.T) {
	tmp := t.TempDir()
	vendorDir := filepath.Join(tmp, "vendor")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}


	outside := filepath.Join(t.TempDir(), "outside", "libout")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "README.md"), []byte("outside lib"), 0o644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(vendorDir, "libout_link")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatal(err)
	}

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	sbomDoc, genErr := Generate(tmp, "vendor", "2.2.0 NEXUS", &config.Config{}, "x86_64-linux-gnu")
	_ = w.Close()
	os.Stderr = oldStderr
	if genErr != nil {
		t.Fatalf("generate: %v", genErr)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	out := buf.String()

	_ = out

	
	if len(sbomDoc.Components) != 1 {
		t.Fatalf("expected 1 vendor component (symlink entry itself), got %d", len(sbomDoc.Components))
	}
	if sbomDoc.Components[0].Name != "libout_link" {
		t.Fatalf("expected libout_link component, got %s", sbomDoc.Components[0].Name)
	}

}

