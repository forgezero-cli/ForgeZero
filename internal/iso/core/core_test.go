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

package core

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"testing"
)

var (
	origLookPath = lookPath
	origCommand  = command
)

func resetGlobals() {
	isoToolPath = ""
	isoToolErr = nil
	isoToolOnce = sync.Once{}
	lookPath = origLookPath
	command = origCommand
}

func hasArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func hasPair(args []string, key, val string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == val {
			return true
		}
	}
	return false
}

func TestDiscoverSuccess(t *testing.T) {
	defer resetGlobals()
	lookPath = func(name string) (string, error) {
		if name == "xorriso" {
			return "/usr/bin/xorriso", nil
		}
		return "", errors.New("not found")
	}
	path, err := Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/usr/bin/xorriso" {
		t.Errorf("expected /usr/bin/xorriso, got %s", path)
	}
}

func TestDiscoverNotFound(t *testing.T) {
	defer resetGlobals()
	lookPath = func(string) (string, error) { return "", errors.New("nope") }
	if _, err := Discover(); err == nil {
		t.Fatal("expected error")
	}
}

func TestDiscoverCache(t *testing.T) {
	defer resetGlobals()
	calls := 0
	lookPath = func(name string) (string, error) {
		calls++
		return "/usr/bin/xorriso", nil
	}
	_, _ = Discover()
	_, _ = Discover()
	if calls != 1 {
		t.Errorf("expected LookPath called once, got %d", calls)
	}
}

func TestBuildEmptySource(t *testing.T) {
	defer resetGlobals()
	if err := Build(Options{}); err == nil {
		t.Fatal("expected error for empty source")
	}
}

func TestBuildNonExistentSource(t *testing.T) {
	defer resetGlobals()
	if err := Build(Options{SourceDir: "/nonexistent/path/xyz"}); err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestBuildSourceIsFile(t *testing.T) {
	defer resetGlobals()
	f, err := os.CreateTemp(t.TempDir(), "f")
	if err != nil {
		t.Fatal(err)
	}
	if err := Build(Options{SourceDir: f.Name()}); err == nil {
		t.Fatal("expected error when source is a file")
	}
}

func TestBuildDiscoverError(t *testing.T) {
	defer resetGlobals()
	lookPath = func(string) (string, error) { return "", errors.New("no tool") }
	if err := Build(Options{SourceDir: t.TempDir()}); err == nil {
		t.Fatal("expected discover error")
	}
}

func TestBuildDefaultsAndSourcePositional(t *testing.T) {
	defer resetGlobals()
	dir := t.TempDir()
	lookPath = func(string) (string, error) { return "xorriso", nil }
	var got []string
	command = func(name string, arg ...string) *exec.Cmd {
		got = arg
		return exec.Command("true")
	}
	if err := Build(Options{SourceDir: dir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasPair(got, "-o", "output.iso") {
		t.Error("expected default -o output.iso")
	}
	if len(got) == 0 || got[len(got)-1] != dir {
		t.Errorf("expected source dir as final positional arg, got %v", got)
	}
}

func TestBuildOutputExtAppended(t *testing.T) {
	defer resetGlobals()
	dir := t.TempDir()
	lookPath = func(string) (string, error) { return "xorriso", nil }
	var got []string
	command = func(name string, arg ...string) *exec.Cmd {
		got = arg
		return exec.Command("true")
	}
	_ = Build(Options{SourceDir: dir, OutputPath: "live"})
	if !hasPair(got, "-o", "live.iso") {
		t.Errorf("expected live.iso, got %v", got)
	}
}

func TestBuildAllOptions(t *testing.T) {
	defer resetGlobals()
	dir := t.TempDir()
	lookPath = func(string) (string, error) { return "xorriso", nil }
	var got []string
	command = func(name string, arg ...string) *exec.Cmd {
		got = arg
		return exec.Command("true")
	}
	opts := Options{
		SourceDir:     dir,
		OutputPath:    "test.iso",
		VolumeID:      "FZ",
		BootImage:     "isolinux.bin",
		BootCatalog:   "boot.cat",
		BootLoadSize:  "4",
		NoEmulBoot:    true,
		BootInfoTable: true,
		Joliet:        true,
		Hybrid:        true,
		CustomArgs:    []string{"-p", "pub"},
	}
	if err := Build(opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"-V", "FZ", "-b", "isolinux.bin", "-c", "boot.cat", "-no-emul-boot", "-boot-info-table", "-J", "-R", "-isohybrid-gpt-basdat", "-p", "pub"} {
		if !hasArg(got, want) {
			t.Errorf("expected arg %q in %v", want, got)
		}
	}
	if !hasPair(got, "-boot-load-size", "4") {
		t.Error("expected -boot-load-size 4")
	}
}

func TestBuildRockRidgeWithoutJoliet(t *testing.T) {
	defer resetGlobals()
	dir := t.TempDir()
	lookPath = func(string) (string, error) { return "xorriso", nil }
	var got []string
	command = func(name string, arg ...string) *exec.Cmd {
		got = arg
		return exec.Command("true")
	}
	_ = Build(Options{SourceDir: dir, OutputPath: "t.iso", RockRidge: true})
	if !hasArg(got, "-R") || hasArg(got, "-J") {
		t.Errorf("expected -R without -J, got %v", got)
	}
}

func TestBuildHybridGenisoimage(t *testing.T) {
	defer resetGlobals()
	dir := t.TempDir()
	lookPath = func(string) (string, error) { return "genisoimage", nil }
	var got []string
	command = func(name string, arg ...string) *exec.Cmd {
		got = arg
		return exec.Command("true")
	}
	_ = Build(Options{SourceDir: dir, OutputPath: "t.iso", Hybrid: true})
	if !hasArg(got, "-isohybrid") {
		t.Errorf("expected -isohybrid, got %v", got)
	}
}

func TestBuildNoHybrid(t *testing.T) {
	defer resetGlobals()
	dir := t.TempDir()
	lookPath = func(string) (string, error) { return "xorriso", nil }
	var got []string
	command = func(name string, arg ...string) *exec.Cmd {
		got = arg
		return exec.Command("true")
	}
	_ = Build(Options{SourceDir: dir, OutputPath: "t.iso"})
	if hasArg(got, "-isohybrid") || hasArg(got, "-isohybrid-gpt-basdat") {
		t.Errorf("unexpected hybrid arg, got %v", got)
	}
}

func TestBuildRunError(t *testing.T) {
	defer resetGlobals()
	dir := t.TempDir()
	lookPath = func(string) (string, error) { return "xorriso", nil }
	command = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("false")
	}
	if err := Build(Options{SourceDir: dir, OutputPath: "t.iso"}); err == nil {
		t.Fatal("expected run error")
	}
}
