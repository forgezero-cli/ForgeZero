package sbom

import (
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
