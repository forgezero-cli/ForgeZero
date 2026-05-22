package audit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fz/internal/config"
)

func TestHasHighSeverity(t *testing.T) {
	r := &Result{Findings: []Finding{{Severity: SeverityLow}}}
	if r.HasHighSeverity() {
		t.Fatal("unexpected high")
	}
	r.Findings = append(r.Findings, Finding{Severity: SeverityHigh})
	if !r.HasHighSeverity() {
		t.Fatal("expected high")
	}
}

func TestScanFileContentGoMod(t *testing.T) {
	tmp := t.TempDir()
	vendor := filepath.Join(tmp, "vendor")
	if err := os.MkdirAll(vendor, 0o755); err != nil {
		t.Fatal(err)
	}
	gomod := filepath.Join(vendor, "go.mod")
	if err := os.WriteFile(gomod, []byte("require openssl v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings := []Finding{}
	seen := map[string]bool{}
	if err := scanFileContent(gomod, &findings, seen); err != nil {
		t.Fatal(err)
	}
	if len(findings) == 0 {
		t.Fatal("expected openssl finding")
	}
}

func TestScanProjectMissingRoot(t *testing.T) {
	_, err := ScanProject(context.Background(), "", "vendor", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScanProjectEscapeRoot(t *testing.T) {
	_, err := ScanProject(context.Background(), "../bad", "vendor", nil)
	if err == nil {
		t.Fatal("expected escape error")
	}
}

func TestScanVendorNotExist(t *testing.T) {
	tmp := t.TempDir()
	findings := []Finding{}
	seen := map[string]bool{}
	if err := scanVendor(context.Background(), tmp, filepath.Join(tmp, "missing"), nil, &findings, seen); err != nil {
		t.Fatal(err)
	}
}

func TestScanVendorNotDirectory(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "file")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings := []Finding{}
	seen := map[string]bool{}
	err := scanVendor(context.Background(), tmp, f, nil, &findings, seen)
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("got %v", err)
	}
}

func TestScanProjectPackageJSON(t *testing.T) {
	tmp := t.TempDir()
	vendor := filepath.Join(tmp, "vendor", "pkg")
	if err := os.MkdirAll(vendor, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendor, "package.json"), []byte(`{"dep":"libcurl"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := ScanProject(context.Background(), tmp, "vendor", nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range result.Findings {
		if f.Package == "cURL" {
			found = true
		}
	}
	if !found {
		t.Fatalf("got %#v", result.Findings)
	}
}

func TestScanProjectCancelledContext(t *testing.T) {
	tmp := t.TempDir()
	vendor := filepath.Join(tmp, "vendor", "deep")
	if err := os.MkdirAll(vendor, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ScanProject(ctx, tmp, "vendor", nil)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestScanSecretsPrivateKey(t *testing.T) {
	tmp := t.TempDir()
	secret := filepath.Join(tmp, "config.env")
	if err := os.WriteFile(secret, []byte("api_key=abcdefghijklmnopqrst"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	findings := []Finding{}
	seen := map[string]bool{}
	if err := scanSecrets(context.Background(), tmp, cfg, &findings, seen); err != nil {
		t.Fatal(err)
	}
}

func TestScanConfigFilesIgnored(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, ".fz.yaml"), []byte("curl http://evil"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{AuditIgnore: []string{".fz.yaml"}}
	findings := []Finding{}
	seen := map[string]bool{}
	if err := scanConfigFiles(tmp, cfg, &findings, seen); err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatal(findings)
	}
}

func TestScanVendorLicensesGPL(t *testing.T) {
	tmp := t.TempDir()
	vendor := filepath.Join(tmp, "vendor", "lib")
	if err := os.MkdirAll(vendor, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendor, "COPYING"), []byte("GNU General Public License"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings := []Finding{}
	seen := map[string]bool{}
	if err := scanVendorLicenses(context.Background(), filepath.Join(tmp, "vendor"), nil, &findings, seen); err != nil {
		t.Fatal(err)
	}
}

func TestScanFileContentReadFail(t *testing.T) {
	findings := []Finding{}
	seen := map[string]bool{}
	err := scanFileContent("/nonexistent/file", &findings, seen)
	if err == nil {
		t.Fatal("expected read error")
	}
}
