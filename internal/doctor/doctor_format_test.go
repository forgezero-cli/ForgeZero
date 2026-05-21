package doctor

import (
	"context"
	"strings"
	"testing"
)

func TestFormatHumanFull(t *testing.T) {
	r := Report{
		Status: "degraded",
		Platform: PlatformReport{
			GOOS: "linux", GOARCH: "amd64", FileSystemImpl: "unix",
			PathSeparator: "/", ExecutionRoot: "/proj", NumCPU: 4,
		},
		Toolchain: []ToolCheck{
			{Name: "gcc", Required: true, Found: true, Path: "/usr/bin/gcc"},
			{Name: "fasm", Required: false, Found: false},
		},
		Permissions: PermReport{
			Root: "/proj", Readable: true, Writable: false,
			DirsScanned: 2, FilesSeen: 5, Error: "partial",
		},
		Errors: []string{"tool missing"},
	}
	out := FormatHuman(r)
	for _, sub := range []string{"degraded", "toolchain:", "gcc", "permissions:", "partial", "tool missing"} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in %s", sub, out)
		}
	}
}

func TestRunEmptyProject(t *testing.T) {
	dir := t.TempDir()
	r, err := Run(context.Background(), Options{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status == "" {
		t.Fatal("empty status")
	}
}
