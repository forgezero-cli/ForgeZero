package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnoreFile(t *testing.T) {
	dir := t.TempDir()
	ignorePath := filepath.Join(dir, ".fzignore")
	content := `
# comment
*.o
temp/
`
	err := os.WriteFile(ignorePath, []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	matcher, err := LoadIgnoreFile(ignorePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(matcher.patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(matcher.patterns))
	}
	if !matcher.Match("test.o") {
		t.Error("*.o should match test.o")
	}
	if !matcher.Match("temp/file.asm") {
		t.Error("temp/ should match temp/file.asm")
	}
}

func TestMatch(t *testing.T) {
	matcher := &IgnoreMatcher{patterns: []string{"*.asm", "test_*", "build/"}}
	if !matcher.Match("main.asm") {
		t.Error("*.asm should match main.asm")
	}
	if !matcher.Match("test_something") {
		t.Error("test_* should match test_something")
	}
	if !matcher.Match("build/output.o") {
		t.Error("build/ should match build/output.o")
	}
	if matcher.Match("main.c") {
		t.Error("main.c should not match")
	}
}
