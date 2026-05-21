package shell

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdBuildNoSourcePath(t *testing.T) {
	state := DefaultState()
	if err := cmdBuild(state); err == nil {
		t.Fatal("expected error")
	}
}

func TestCmdSetUnknownKey(t *testing.T) {
	state := DefaultState()
	if err := cmdSet(state, []string{"set", "unknown=x"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestCmdSetMissingValue(t *testing.T) {
	state := DefaultState()
	if err := cmdSet(state, []string{"set"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestCmdBuildDirNoSource(t *testing.T) {
	state := DefaultState()
	state.SourceType = "dir"
	state.SourcePath = filepath.Join(t.TempDir(), "empty")
	state.Out = filepath.Join(t.TempDir(), "out")
	if err := cmdBuild(state); err == nil {
		t.Fatal("expected build error")
	}
}

func TestCmdShowAllFields(t *testing.T) {
	state := DefaultState()
	state.Sanitize = true
	state.NoCache = true
	state.LdScript = "x.ld"
	state.TextAddr = "0x0"
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	cmdShow(state)
	w.Close()
	os.Stdout = old
}
