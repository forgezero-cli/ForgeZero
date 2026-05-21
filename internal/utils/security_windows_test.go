package utils

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateCLIPathWindowsDrive(t *testing.T) {
	cases := []string{
		`C:\project\src\main.c`,
		`D:\build\out.exe`,
		`\\server\share\vendor\lib`,
	}
	for _, p := range cases {
		if err := ValidateCLIPath(p); err != nil {
			t.Fatalf("%q: %v", p, err)
		}
	}
}

func TestValidateCLIPathWindowsTraversal(t *testing.T) {
	cases := []string{
		`C:\proj\..\windows\system32`,
		`..\secret`,
	}
	for _, p := range cases {
		if err := ValidateCLIPath(p); err == nil {
			t.Fatalf("%q expected error", p)
		}
	}
}

func TestPathWithinRootDrive(t *testing.T) {
	root := `C:\project`
	inside := `C:\project\src\a.c`
	if !pathWithinRoot(root, inside) {
		t.Fatal("expected inside")
	}
	outside := `D:\other\a.c`
	if pathWithinRoot(root, outside) {
		t.Fatal("expected outside")
	}
}

func TestSetExecutionRootClean(t *testing.T) {
	prev := GetExecutionRoot()
	SetExecutionRoot(`C:\build\`)
	if GetExecutionRoot() != filepath.Clean(`C:\build\`) {
		t.Fatalf("got %q", GetExecutionRoot())
	}
	SetExecutionRoot(prev)
}

func TestForbiddenPathCharsWindowsAllowsBackslash(t *testing.T) {
	if runtime.GOOS != "windows" {
		chars := forbiddenPathChars()
		if chars == forbiddenArgChars() {
			t.Log("non-windows build uses unix rules")
		}
		return
	}
	if !containsChar(`C:\path`, '\\') {
		t.Fatal("backslash required in test path")
	}
	if err := ValidateCLIPath(`C:\valid\path.txt`); err != nil {
		t.Fatal(err)
	}
}

func containsChar(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}

func TestResolveSecurePathDrive(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("drive letter resolution requires windows")
	}
	dir := t.TempDir()
	resolved, err := ResolveSecurePath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if resolved == "" {
		t.Fatal("empty")
	}
}

func TestIsUnsafeUNC(t *testing.T) {
	if !isUnsafeUNC(`\\`) {
		t.Fatal("bare UNC should be unsafe")
	}
	if isUnsafeUNC(`\\server\share`) {
		t.Fatal("valid UNC should be allowed")
	}
}
