package fs

import (
	"path/filepath"
	"testing"
)

func TestCleanPathDrive(t *testing.T) {
	got := CleanPath(`C:\project\src\main.go`)
	if got == "" {
		t.Fatal("empty path")
	}
	if !HasDrivePrefix(got) {
		t.Fatalf("expected drive prefix in %q", got)
	}
	if filepath.Base(filepath.FromSlash(`C:/project/src/main.go`)) != "main.go" {
		t.Fatal("sanity check failed")
	}
}

func TestCleanPathUNC(t *testing.T) {
	unc := `\\server\share\dir\file.txt`
	got := CleanPath(unc)
	if !IsUNC(got) {
		t.Fatalf("expected UNC, got %q", got)
	}
}

func TestHasDrivePrefix(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{`D:\build`, true},
		{`/unix`, false},
		{`C:relative`, true},
	}
	for _, tc := range cases {
		if HasDrivePrefix(tc.path) != tc.want {
			t.Fatalf("%q: got %v want %v", tc.path, !tc.want, tc.want)
		}
	}
}

func TestNormalizeAbsRelative(t *testing.T) {
	dir := t.TempDir()
	abs, err := NormalizeAbs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if abs == "" {
		t.Fatal("empty abs")
	}
}
