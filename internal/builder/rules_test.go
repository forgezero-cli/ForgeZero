/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version of the License.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package builder

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/forgezero-cli/ForgeZero/internal/config"
)

func TestBuildRulesGraph(t *testing.T) {
	rules := []config.BuildRule{
		{Name: "gen", Action: "printf hi > $out", Outputs: []string{"out.txt"}},
		{Name: "copy", Action: "cat $in > $out", Inputs: []string{"out.txt"}, Outputs: []string{"dup.txt"}},
	}
	graph, err := buildRulesGraph(rules)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(graph))
	}
	if len(graph[0]) != 0 {
		t.Fatalf("expected first node to have no deps, got %v", graph[0])
	}
	if len(graph[1]) != 1 || graph[1][0] != 0 {
		t.Fatalf("expected second node to depend on first, got %v", graph[1])
	}
}

func TestRunBuildRules(t *testing.T) {
	dir := t.TempDir()
	out1 := filepath.Join(dir, "generated.txt")
	out2 := filepath.Join(dir, "final.txt")
	rules := []config.BuildRule{
		{Name: "generate", Action: "printf hello > $out", Outputs: []string{out1}},
		{Name: "copy", Action: "cat $in > $out", Inputs: []string{out1}, Outputs: []string{out2}},
	}
	cfg := &config.Config{BuildRules: rules}
	res, err := runBuildRules(context.Background(), cfg, true, 2)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if len(res.ObjectFiles) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(res.ObjectFiles))
	}
	if _, err := os.Stat(out1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out2); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out2)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "hello" {
		t.Fatalf("unexpected final content: %q", string(data))
	}
}

func TestRunBuildRulesDepfileRestat(t *testing.T) {
	dir := t.TempDir()
	dep1 := filepath.Join(dir, "a.txt")
	dep2 := filepath.Join(dir, "b.txt")
	out := filepath.Join(dir, "out.txt")
	depfile := filepath.Join(dir, "out.d")
	if err := os.WriteFile(dep1, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dep2, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	d1 := time.Now().Add(-2 * time.Hour)
	d2 := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(out, d1, d1); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(depfile, []byte("out.txt: a.txt b.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(depfile, d2, d2); err != nil {
		t.Fatal(err)
	}
	rule := config.BuildRule{
		Name:    "build",
		Action:  "cat $in > $out && printf '%s: %s %s\n' $out $in > $depfile",
		Inputs:  []string{dep1, dep2},
		Outputs: []string{out},
		Depfile: depfile,
	}
	cfg := &config.Config{BuildRules: []config.BuildRule{rule}}
	if _, err := runBuildRules(context.Background(), cfg, true, 1); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.ModTime().Before(d2) {
		t.Fatal("output did not update")
	}
	now := time.Now()
	if err := os.Chtimes(out, now, now); err != nil {
		t.Fatal(err)
	}
	cfg.BuildRules[0].Action = "false"
	if _, err := runBuildRules(context.Background(), cfg, true, 1); err != nil {
		t.Fatal(err)
	}
}
