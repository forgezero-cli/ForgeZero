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

package compilecommands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/config"
)

func TestGenerateWithSpacesAndMixedFiles(t *testing.T) {
	dir := t.TempDir()
	spaceDir := filepath.Join(dir, "sub dir")
	if err := os.MkdirAll(spaceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cFile := filepath.Join(spaceDir, "main.c")
	if err := os.WriteFile(cFile, []byte("int main() { return 0; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		SourceFiles: []string{cFile},
	}
	oldWd, _ := os.Getwd()
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := Generate(cfg, "."); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("compile_commands.json")
	if err != nil {
		t.Fatal(err)
	}
	var cmds []CompileCommand
	if err := json.Unmarshal(data, &cmds); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(cmds) != 1 {
		t.Errorf("expected 1 command, got %d", len(cmds))
	}
	if !strings.Contains(string(data), "sub dir") {
		t.Error("path with space not preserved")
	}
}
