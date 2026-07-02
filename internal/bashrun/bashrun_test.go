package bashrun

import (
	"context"
	"os"
	"testing"
)

func TestRunInlineSystemShell(t *testing.T) {
    if err := RunInline(context.Background(), "echo hi", false); err != nil {
        t.Fatalf("RunInline failed: %v", err)
    }
}

func TestRunInlineInternalFallback(t *testing.T) {
    old := os.Getenv("PATH")
    defer os.Setenv("PATH", old)
    os.Setenv("PATH", "")
    if err := RunInline(context.Background(), "export FZTEST=1\ncd .", false); err != nil {
        t.Fatalf("internal RunInline failed: %v", err)
    }
}
