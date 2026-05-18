package shell

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{`build`, []string{"build"}},
		{`set mode=raw`, []string{"set", "mode=raw"}},
		{`set "ld-script=linker.ld"`, []string{"set", "ld-script=linker.ld"}},
		{`show`, []string{"show"}},
		{`exit`, []string{"exit"}},
	}
	for _, tt := range tests {
		got := splitCommand(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitCommand(%q) = %v, want %v", tt.input, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitCommand(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestCmdHelp(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cmdHelp()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "Commands:") {
		t.Error("help output missing 'Commands:'")
	}
}

func TestCmdShow(t *testing.T) {
	state := DefaultState()
	state.Mode = "raw"
	state.Format = "bin"
	state.Strict = true
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cmdShow(state)
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "Mode: raw") {
		t.Error("show output missing Mode")
	}
	if !strings.Contains(out, "Format: bin") {
		t.Error("show output missing Format")
	}
}

func TestCmdSet(t *testing.T) {
	state := DefaultState()
	args := []string{"set", "mode=raw"}
	if err := cmdSet(state, args); err != nil {
		t.Fatal(err)
	}
	if state.Mode != "raw" {
		t.Errorf("expected mode raw, got %s", state.Mode)
	}
	args = []string{"set", "strict=true"}
	if err := cmdSet(state, args); err != nil {
		t.Fatal(err)
	}
	if !state.Strict {
		t.Error("expected strict true")
	}
	args = []string{"set", "invalid"}
	if err := cmdSet(state, args); err == nil {
		t.Error("expected error for invalid set syntax")
	}
}
