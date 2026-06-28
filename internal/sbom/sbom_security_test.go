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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func TestGenerateWasmTarget(t *testing.T) {
	root := t.TempDir()
	sb, err := Generate(root, "vendor", "1.0.0", &config.Config{}, "wasm32-wasi")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tool := range sb.Metadata.Tools {
		if tool.Name == "wasm-target" {
			found = true
		}
	}
	if !found {
		t.Error("wasm-target tool not in SBOM metadata")
	}
}

func TestScanVendorSecureHash(t *testing.T) {
	root := t.TempDir()
	vendor := filepath.Join(root, "vendor")
	pkg := filepath.Join(vendor, "lib")
	if err := os.MkdirAll(pkg, utils.DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "README.md"), []byte("pkg"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	components, err := scanVendorComponents(root, "vendor")
	if err != nil {
		t.Fatal(err)
	}
	if len(components) != 1 {
		t.Fatalf("components = %d, want 1", len(components))
	}
	if components[0].Hashes[0].Algorithm != "BLAKE3" {
		t.Errorf("algorithm = %s", components[0].Hashes[0].Algorithm)
	}
}

func TestQueryToolVersionMissing(t *testing.T) {
	_, ok := queryToolVersion("_fz_missing_tool_xyz_", "--version")
	if ok {
		t.Error("expected missing tool")
	}
}

func TestMarshalSBOM(t *testing.T) {
	sb, err := Generate(t.TempDir(), "vendor", "2.0", nil, "x86_64-linux-gnu")
	if err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(sb)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty SBOM JSON")
	}
}

func TestDetectToolchainWithContext(t *testing.T) {
	_ = context.Background()
	tools := detectToolchainVersions("wasm32-wasi")
	if len(tools) == 0 {
		t.Log("no toolchain tools in PATH (acceptable in CI)")
	}
}
