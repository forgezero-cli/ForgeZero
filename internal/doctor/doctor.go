/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	fzvfs "github.com/forgezero-cli/ForgeZero/internal/fs"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
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
			report.Errors = append(report.Errors, "doctor panic: "+toString(r))
			err = errors.New("doctor panic: " + toString(r))
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

func toString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case error:
		return x.Error()
	default:
		return "unknown panic"
	}
}

func MarshalJSON(r Report) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func (r *Report) healthyFromChecks() {
	for _, t := range r.Toolchain {
		if t.Required && !t.Found {
			r.Healthy = false
			r.Errors = append(r.Errors, "toolchain: "+t.Name+" unavailable")
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
			pr.Error = "permissions panic: " + toString(r)
			pr.Readable = false
			pr.Writable = false
		}
	}()
	resolved, err := utils.ResolveSecurePath(root)
	if err != nil {
		pr.Error = "resolve root: " + err.Error()
		return pr
	}
	pr.Root = resolved
	info, err := utils.StatResolved(resolved)
	if err != nil {
		pr.Error = "stat root: " + err.Error()
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
		return errors.New("write probe: " + err.Error())
	}
	if err := utils.RemovePath(probe); err != nil {
		return errors.New("remove probe: " + err.Error())
	}
	return nil
}

func scanTree(root string) (dirs, files int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("scan panic: " + toString(r))
		}
	}()
	stack := []string{root}
	for len(stack) > 0 {
		dir := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		dirs++
		entries, rdErr := utils.ReadDirResolved(dir)
		if rdErr != nil {
			return dirs, files, errors.New("readdir " + dir + ": " + rdErr.Error())
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
				return dirs, files, errors.New("lstat " + path + ": " + lerr.Error())
			}
			if info.Mode()&os.ModeSymlink != 0 {
				continue
			}
			f, oerr := utils.OpenVerifiedRead(path)
			if oerr != nil {
				return dirs, files, errors.New("read " + path + ": " + oerr.Error())
			}
			_ = f.Close()
		}
	}
	return dirs, files, nil
}

func FormatHuman(r Report) string {
	var b strings.Builder
	b.WriteString("fz doctor: ")
	b.WriteString(r.Status)
	b.WriteString("\n")
	b.WriteString("platform: ")
	b.WriteString(r.Platform.GOOS)
	b.WriteString("/")
	b.WriteString(r.Platform.GOARCH)
	b.WriteString(" fs=")
	b.WriteString(r.Platform.FileSystemImpl)
	b.WriteString(" sep=")
	b.WriteString(r.Platform.PathSeparator)
	b.WriteString(" root=")
	b.WriteString(r.Platform.ExecutionRoot)
	b.WriteString(" cpus=")
	b.WriteString(strconv.Itoa(r.Platform.NumCPU))
	b.WriteString("\n")
	b.WriteString("toolchain:\n")
	for _, t := range r.Toolchain {
		b.WriteString("  ")
		b.WriteString(t.Name)
		if t.Required {
			b.WriteString(" (required)")
		}
		b.WriteString(": ")
		if t.Found {
			b.WriteString(t.Path)
		} else {
			b.WriteString("missing")
		}
		b.WriteString("\n")
	}
	b.WriteString("permissions: root=")
	b.WriteString(r.Permissions.Root)
	b.WriteString(" readable=")
	b.WriteString(strconv.FormatBool(r.Permissions.Readable))
	b.WriteString(" writable=")
	b.WriteString(strconv.FormatBool(r.Permissions.Writable))
	b.WriteString(" dirs=")
	b.WriteString(strconv.Itoa(r.Permissions.DirsScanned))
	b.WriteString(" files=")
	b.WriteString(strconv.Itoa(r.Permissions.FilesSeen))
	b.WriteString("\n")
	if r.Permissions.Error != "" {
		b.WriteString("  error: ")
		b.WriteString(r.Permissions.Error)
		b.WriteString("\n")
	}
	for _, e := range r.Errors {
		b.WriteString("issue: ")
		b.WriteString(e)
		b.WriteString("\n")
	}
	return b.String()
}
