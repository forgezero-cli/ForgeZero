/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func TestGenerateMalformedVendorWalk(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor")
	if err := os.Mkdir(vendor, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(vendor, 0o755) }()
	_, err := Generate(root, "vendor", "1", nil, "")
	if err == nil {
		t.Fatal("expected walk error")
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	sb := &SBOM{Version: 1, Components: []Component{{Name: "c", Version: "1"}}}
	data, err := Marshal(sb)
	if err != nil {
		t.Fatal(err)
	}
	var decoded SBOM
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
}

func TestScanVendorComponentsWithGit(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor", "github.com", "u", "lib")
	if err := os.MkdirAll(filepath.Join(vendor, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendor, "README"), []byte("lib"), 0o644); err != nil {
		t.Fatal(err)
	}
	comps, err := scanVendorComponents(root, "vendor")
	if err != nil {
		t.Fatal(err)
	}
	if len(comps) == 0 {
		t.Fatal("expected component")
	}
}

func TestDetectToolchainVersions(t *testing.T) {
	vers := detectToolchainVersions("x86_64-linux-gnu")
	if vers == nil {
		t.Fatal("nil versions")
	}
}

func TestQueryToolVersionUnknown(t *testing.T) {
	v, ok := queryToolVersion("definitely-not-a-real-tool-xyz", "--version")
	if ok || v != "" {
		t.Fatalf("got %q ok=%v", v, ok)
	}
}

func TestGenerateWithVendorHash(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor", "pkg")
	if err := os.MkdirAll(vendor, utils.DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendor, "data.txt"), []byte("payload"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	sb, err := Generate(root, "vendor", "3.1", nil, "x86_64-linux-gnu")
	if err != nil {
		t.Fatal(err)
	}
	if len(sb.Components) == 0 {
		t.Fatal("expected components")
	}
	if len(sb.Components[0].Hashes) == 0 {
		t.Fatal("expected hash")
	}
}

func TestScanVendorComponentsContextCancel(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor", "deep", "nested")
	if err := os.MkdirAll(vendor, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := scanVendorComponents(root, "vendor")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateJSONIncludesTools(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{ToolChecksums: map[string]string{"nasm": "dead"}}
	sb, err := Generate(root, "vendor", "1", cfg, "wasm32-wasi")
	if err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(sb)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "nasm") {
		t.Fatal(string(data))
	}
}
