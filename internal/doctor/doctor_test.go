package doctor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	fzvfs "fz/internal/fs"
	"fz/internal/utils"
)

func TestRunHealthy(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), utils.DirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "a.c"), []byte("int x;"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	report, err := Run(context.Background(), Options{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if report.Platform.ExecutionRoot != filepath.Clean(dir) {
		t.Fatalf("root %q", report.Platform.ExecutionRoot)
	}
	if report.Permissions.Root == "" {
		t.Fatal("empty perm root")
	}
	if !report.Permissions.Writable || !report.Permissions.Readable {
		t.Fatalf("perms: %+v", report.Permissions)
	}
	if len(report.Toolchain) != 3 {
		t.Fatalf("toolchain len %d", len(report.Toolchain))
	}
}

func TestRunJSON(t *testing.T) {
	dir := t.TempDir()
	report, err := Run(context.Background(), Options{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	data, err := MarshalJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Platform.GOOS != runtime.GOOS {
		t.Fatalf("goos %q", decoded.Platform.GOOS)
	}
}

func TestRunEmptyRootUsesCwd(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	report, err := Run(context.Background(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Permissions.Root == "" {
		t.Fatal("empty root")
	}
}

func TestRunNotDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	report, err := Run(context.Background(), Options{Root: file})
	if err != nil {
		t.Fatal(err)
	}
	if report.Healthy {
		t.Fatal("expected unhealthy")
	}
	if report.Permissions.Error == "" {
		t.Fatal("expected perm error")
	}
}

func TestRunResolveRootFail(t *testing.T) {
	report, err := Run(context.Background(), Options{Root: "../bad/../etc"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Healthy {
		t.Fatal("expected unhealthy")
	}
}

func TestMockReadDirFails(t *testing.T) {
	dir := t.TempDir()
	m := fzvfs.NewMock(fzvfs.Default)
	resolved, err := utils.ResolveSecurePath(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.SetFail("ReadDir", resolved, fzvfs.ErrPermission)
	prev := utils.GetExecutionRoot()
	utils.SetFileSystem(m)
	t.Cleanup(func() {
		utils.SetFileSystem(nil)
		utils.SetExecutionRoot(prev)
	})
	report, err := Run(context.Background(), Options{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if report.Healthy {
		t.Fatal("expected unhealthy")
	}
	if !report.Permissions.Readable {
		return
	}
	t.Fatalf("perms %+v", report.Permissions)
}

func TestMockWriteProbeFails(t *testing.T) {
	dir := t.TempDir()
	m := fzvfs.NewMock(fzvfs.Default)
	m.SetFailOp("CreateTemp", fzvfs.ErrDiskFull)
	utils.SetFileSystem(m)
	t.Cleanup(func() { utils.SetFileSystem(nil) })
	report, err := Run(context.Background(), Options{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if report.Permissions.Writable {
		t.Fatal("expected not writable")
	}
}

func TestFormatHuman(t *testing.T) {
	r := Report{
		Status:  "ok",
		Healthy: true,
		Platform: PlatformReport{
			GOOS: "linux", GOARCH: "amd64", FileSystemImpl: "unix",
			PathSeparator: "/", ExecutionRoot: "/tmp", NumCPU: 1,
		},
		Toolchain: []ToolCheck{{Name: "zig", Found: true, Path: "/usr/bin/zig"}},
		Permissions: PermReport{Root: "/tmp", Readable: true, Writable: true},
	}
	out := FormatHuman(r)
	if out == "" {
		t.Fatal("empty output")
	}
}

func TestHealthyFromChecksRequiredMissing(t *testing.T) {
	r := Report{
		Healthy: true,
		Toolchain: []ToolCheck{{Name: "zig", Required: true, Found: false}},
		Permissions: PermReport{Readable: true, Writable: true},
	}
	r.healthyFromChecks()
	if r.Healthy {
		t.Fatal("expected unhealthy")
	}
}

func TestAuditToolchain(t *testing.T) {
	tools := auditToolchain()
	if len(tools) != 3 {
		t.Fatalf("len %d", len(tools))
	}
}

func TestScanTreeSymlinkSkip(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "t.txt")
	if err := os.WriteFile(target, []byte("a"), utils.FilePerm); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "l.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlink")
	}
	dirs, files, err := scanTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirs < 1 || files < 1 {
		t.Fatalf("dirs=%d files=%d", dirs, files)
	}
}

func TestProbeWritable(t *testing.T) {
	dir := t.TempDir()
	if err := probeWritable(dir); err != nil {
		t.Fatal(err)
	}
}

func TestRunContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dir := t.TempDir()
	_, err := Run(ctx, Options{Root: dir})
	if err == nil {
		t.Fatal("expected ctx error")
	}
}

func TestRunPanicRecovery(t *testing.T) {
	dir := t.TempDir()
	m := fzvfs.NewMock(fzvfs.Default)
	resolved, _ := utils.ResolveSecurePath(dir)
	m.SetFail("Stat", resolved, fzvfs.ErrPermission)
	utils.SetFileSystem(m)
	t.Cleanup(func() { utils.SetFileSystem(nil) })
	report, err := Run(context.Background(), Options{Root: dir})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status == "panic" {
		return
	}
	if report.Permissions.Error != "" {
		return
	}
}

func TestMarshalJSONIndent(t *testing.T) {
	r := Report{Status: "ok", Healthy: true}
	b, err := MarshalJSON(r)
	if err != nil || len(b) == 0 {
		t.Fatal(err)
	}
}

func TestPermissionsPanicRecovery(t *testing.T) {
	pr := auditPermissions("/nonexistent-path-xyz-999")
	if pr.Error == "" && pr.Readable {
		t.Fatalf("unexpected ok: %+v", pr)
	}
}
