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

package linker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func TestDedupObjectsEmpty(t *testing.T) {
	got := dedupObjects(nil)
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestDedupObjectsUnique(t *testing.T) {
	got := dedupObjects([]string{"a.o", "b.o", "a.o"})
	if len(got) != 2 {
		t.Fatalf("expected 2 unique objects, got %d", len(got))
	}
	if got[0] != "a.o" || got[1] != "b.o" {
		t.Fatalf("unexpected dedup result %v", got)
	}
}

func TestBuildLinkCommandRawModeNoConfig(t *testing.T) {
	cfg := &config.Config{Mode: "raw"}
	cmd, args, err := buildLinkCommand([]string{"a.o"}, "out", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cmd != ldForTarget() {
		t.Fatalf("expected %s, got %s", ldForTarget(), cmd)
	}
	if len(args) != 3 || args[0] != "a.o" || args[1] != "-o" || args[2] != "out" {
		t.Fatalf("unexpected args %v", args)
	}
}

func TestBuildLinkCommandZigRequestedUnavailable(t *testing.T) {
	oldCheck := utils.CheckToolFunc
	utils.CheckToolFunc = func(string) error { return errors.New("zig missing") }
	defer func() { utils.CheckToolFunc = oldCheck }()

	old := PreferredLinker
	PreferredLinker = ""
	defer func() { PreferredLinker = old }()

	cfg := &config.Config{Mode: "c", Toolchain: "zig"}
	_, _, err := buildLinkCommand([]string{"a.o"}, "out", cfg)
	if err == nil {
		t.Fatal("expected error when zig requested but unavailable")
	}
}

func TestHasUndefinedSymbolOutput(t *testing.T) {
	if !hasUndefinedSymbol("undefined reference to foo") {
		t.Fatal("expected undefined symbol detection")
	}
	if hasUndefinedSymbol("") {
		t.Fatal("expected false for empty output")
	}
}

func TestBuildLinkCommandHandlesLargeArgumentLists(t *testing.T) {
	cfg := &config.Config{Mode: "c", Toolchain: "c"}
	objs := make([]string, 300)
	for i := range objs {
		objs[i] = fmt.Sprintf("file%d.o", i)
	}
	cmd, args, err := buildLinkCommand(objs, "out", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == "" {
		t.Fatal("expected non-empty linker command")
	}
	if len(args) == 0 {
		t.Fatal("expected args")
	}
}

func TestLinkObjectsReportsUndefinedSymbols(t *testing.T) {
	oldCheck := utils.CheckToolFunc
	oldRunner := runner
	utils.CheckToolFunc = func(string) error { return nil }
	runner = &MockRunner{RunFunc: func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		return "undefined reference to bar", errors.New("link failed")
	}}
	defer func() {
		utils.CheckToolFunc = oldCheck
		runner = oldRunner
	}()
	if err := LinkObjects(context.Background(), "out", []string{"a.o"}, &config.Config{Mode: "raw"}); err == nil || !strings.Contains(err.Error(), "undefined symbols") {
		t.Fatalf("expected undefined symbol error, got %v", err)
	}
}
