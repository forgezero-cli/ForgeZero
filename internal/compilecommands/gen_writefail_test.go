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
	"os"
	"path/filepath"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	fzvfs "github.com/forgezero-cli/ForgeZero/internal/fs"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func TestGenerateSecureWriteFail(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	c := filepath.Join(dir, "m.c")
	if err := os.WriteFile(c, []byte("int main(){}"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFailOp("Rename", fzvfs.ErrDiskFull)
	utils.SetFileSystem(m)
	defer utils.SetFileSystem(nil)
	cfg := &config.Config{SourceFiles: []string{c}}
	if err := Generate(cfg, dir); err == nil {
		t.Fatal("expected write error")
	}
}
