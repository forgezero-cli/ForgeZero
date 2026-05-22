package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	fzvfs "fz/internal/fs"
	"fz/internal/utils"
)

type Options struct {
	Root string
}

type ToolCheck struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Found    bool   `json:"found"`
	Path     string `json:"path,omitempty"`
	Error    string `json:"error,omitempty"`
}

type PermReport struct {
	Root        string `json:"root"`
	Writable    bool   `json:"writable"`
	Readable    bool   `json:"readable"`
	DirsScanned int    `json:"dirs_scanned"`
	FilesSeen   int    `json:"files_seen"`
	Error       string `json:"error,omitempty"`
}

type PlatformReport struct {
	GOOS           string `json:"goos"`
	GOARCH         string `json:"goarch"`
	PathSeparator  string `json:"path_separator"`
	FileSystemImpl string `json:"filesystem_impl"`
	ExecutionRoot  string `json:"execution_root"`
	NumCPU         int    `json:"num_cpu"`
}

type Report struct {
	Status      string         `json:"status"`
	Healthy     bool           `json:"healthy"`
	Toolchain   []ToolCheck    `json:"toolchain"`
	Permissions PermReport     `json:"permissions"`
	Platform    PlatformReport `json:"platform"`
	Errors      []string       `json:"errors,omitempty"`
}

func Run(ctx context.Context, opts Options) (report Report, err error) {
	defer func() {
		if r := recover(); r != nil {
			report = Report{Status: "panic", Healthy: false}
			report.Errors = append(report.Errors, fmt.Sprintf("doctor panic: %v", r))
			err = fmt.Errorf("doctor panic: %v", r)
		}
	}()
	if ctx == nil {
		ctx = context.Background()
	}
	root := opts.Root
	if root == "" {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return Report{Status: "error", Healthy: false, Errors: []string{cwdErr.Error()}}, cwdErr
		}
		root = cwd
	}
	root = filepath.Clean(root)
	utils.SetExecutionRoot(root)
	report = Report{
		Status:  "ok",
		Healthy: true,
		Platform: PlatformReport{
			GOOS:           runtime.GOOS,
			GOARCH:         runtime.GOARCH,
			PathSeparator:  string(os.PathSeparator),
			FileSystemImpl: fzvfs.ImplName(),
			ExecutionRoot:  utils.GetExecutionRoot(),
			NumCPU:         runtime.NumCPU(),
		},
	}
	report.Toolchain = auditToolchain()
	report.Permissions = auditPermissions(root)
	report.healthyFromChecks()
	if !report.Healthy {
		report.Status = "degraded"
	}
	select {
	case <-ctx.Done():
		return report, ctx.Err()
	default:
		return report, nil
	}
}

func MarshalJSON(r Report) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func (r *Report) healthyFromChecks() {
	for _, t := range r.Toolchain {
		if t.Required && !t.Found {
			r.Healthy = false
			r.Errors = append(r.Errors, fmt.Sprintf("toolchain: %s unavailable", t.Name))
		}
	}
	if !r.Permissions.Readable || !r.Permissions.Writable {
		r.Healthy = false
		if r.Permissions.Error != "" {
			r.Errors = append(r.Errors, r.Permissions.Error)
		}
	}
}

func auditToolchain() []ToolCheck {
	specs := []struct {
		name     string
		required bool
	}{
		{name: "zig", required: true},
		{name: "fasm", required: runtime.GOOS != "windows"},
		{name: "wasm-ld", required: false},
	}
	out := make([]ToolCheck, 0, len(specs))
	for _, spec := range specs {
		tc := ToolCheck{Name: spec.name, Required: spec.required}
		path, lookErr := utils.LookExecutable(spec.name)
		if lookErr == nil {
			tc.Found = true
			tc.Path = path
		} else {
			tc.Error = lookErr.Error()
		}
		out = append(out, tc)
	}
	return out
}

func auditPermissions(root string) PermReport {
	pr := PermReport{Root: root}
	defer func() {
		if r := recover(); r != nil {
			pr.Error = fmt.Sprintf("permissions panic: %v", r)
			pr.Readable = false
			pr.Writable = false
		}
	}()
	resolved, err := utils.ResolveSecurePath(root)
	if err != nil {
		pr.Error = fmt.Sprintf("resolve root: %v", err)
		return pr
	}
	pr.Root = resolved
	info, err := utils.StatResolved(resolved)
	if err != nil {
		pr.Error = fmt.Sprintf("stat root: %v", err)
		return pr
	}
	if !info.IsDir() {
		pr.Error = "execution root is not a directory"
		return pr
	}
	pr.Readable = true
	if err := probeWritable(resolved); err != nil {
		pr.Error = err.Error()
		pr.Writable = false
	} else {
		pr.Writable = true
	}
	dirs, files, walkErr := scanTree(resolved)
	pr.DirsScanned = dirs
	pr.FilesSeen = files
	if walkErr != nil {
		pr.Readable = false
		if pr.Error == "" {
			pr.Error = walkErr.Error()
		} else {
			pr.Error = pr.Error + "; " + walkErr.Error()
		}
	}
	return pr
}

func probeWritable(root string) error {
	probe := filepath.Join(root, ".fz_doctor_probe")
	data := []byte("probe")
	if err := utils.SecureWriteFile(probe, data); err != nil {
		return fmt.Errorf("write probe: %w", err)
	}
	if err := utils.RemovePath(probe); err != nil {
		return fmt.Errorf("remove probe: %w", err)
	}
	return nil
}

func scanTree(root string) (dirs, files int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("scan panic: %v", r)
		}
	}()
	stack := []string{root}
	for len(stack) > 0 {
		dir := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		dirs++
		entries, rdErr := utils.ReadDirResolved(dir)
		if rdErr != nil {
			return dirs, files, fmt.Errorf("readdir %s: %w", dir, rdErr)
		}
		for _, ent := range entries {
			name := ent.Name()
			if name == ".git" || name == ".fz_objs" || name == ".fz_cache" || name == "vendor" {
				continue
			}
			path := filepath.Join(dir, name)
			if ent.IsDir() {
				stack = append(stack, path)
				continue
			}
			files++
			info, lerr := utils.LstatPath(path)
			if lerr != nil {
				return dirs, files, fmt.Errorf("lstat %s: %w", path, lerr)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				continue
			}
			f, oerr := utils.OpenVerifiedRead(path)
			if oerr != nil {
				return dirs, files, fmt.Errorf("read %s: %w", path, oerr)
			}
			f.Close()
		}
	}
	return dirs, files, nil
}

func FormatHuman(r Report) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("fz doctor: %s\n", r.Status))
	b.WriteString(fmt.Sprintf("platform: %s/%s fs=%s sep=%q root=%s cpus=%d\n",
		r.Platform.GOOS, r.Platform.GOARCH, r.Platform.FileSystemImpl,
		r.Platform.PathSeparator, r.Platform.ExecutionRoot, r.Platform.NumCPU))
	b.WriteString("toolchain:\n")
	for _, t := range r.Toolchain {
		state := "missing"
		if t.Found {
			state = t.Path
		}
		req := ""
		if t.Required {
			req = " (required)"
		}
		b.WriteString(fmt.Sprintf("  %s%s: %s\n", t.Name, req, state))
	}
	b.WriteString(fmt.Sprintf("permissions: root=%s readable=%v writable=%v dirs=%d files=%d\n",
		r.Permissions.Root, r.Permissions.Readable, r.Permissions.Writable,
		r.Permissions.DirsScanned, r.Permissions.FilesSeen))
	if r.Permissions.Error != "" {
		b.WriteString(fmt.Sprintf("  error: %s\n", r.Permissions.Error))
	}
	for _, e := range r.Errors {
		b.WriteString(fmt.Sprintf("issue: %s\n", e))
	}
	return b.String()
}
