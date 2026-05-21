package utils

import (
	"os/exec"
	"runtime"
	"testing"
)

func TestLookExecutableGo(t *testing.T) {
	path, err := lookExecutable("go")
	if err != nil {
		t.Skip("go not in PATH")
	}
	if path == "" {
		t.Fatal("empty path")
	}
}

func TestLookExecutableWindowsExe(t *testing.T) {
	if runtime.GOOS != "windows" {
		name := "go"
		path, err := lookExecutable(name)
		if err != nil {
			t.Skip("go missing")
		}
		_, err2 := exec.LookPath(name)
		if err2 != nil {
			t.Skip("go missing")
		}
		if path == "" {
			t.Fatal("empty")
		}
		return
	}
	path, err := lookExecutable("cmd")
	if err != nil {
		t.Skip("cmd missing")
	}
	if path == "" {
		t.Fatal("empty path")
	}
}
