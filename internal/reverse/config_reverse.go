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

package reverse

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/config"
)

type ReverseConfig struct {
	Name    string
	Target  string
	Sysroot string
	Source  []string
	Output  string
	Libs    []string
	Include []string
	Flags   []string
	Defines map[string]string
}

func ReverseMakefile(filename string) (*config.Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rc := &ReverseConfig{
		Source:  make([]string, 0, 16),
		Libs:    make([]string, 0, 8),
		Include: make([]string, 0, 8),
		Flags:   make([]string, 0, 16),
		Defines: make(map[string]string),
	}

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 65536)
	scanner.Buffer(buf, 65536)

	var inTarget bool
	var targetName string
	var targetDeps []string
	var targetCmds []string

	for scanner.Scan() {
		line := scanner.Bytes()
		line = trimBytes(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		if idx := bytes.IndexByte(line, '='); idx > 0 {
			key := trimBytes(line[:idx])
			val := trimBytes(line[idx+1:])
			rc.parseVariable(key, val)
			continue
		}

		if idx := bytes.IndexByte(line, ':'); idx > 0 && !bytes.Contains(line, []byte(":=")) {
			inTarget = true
			targetName = string(trimBytes(line[:idx]))
			deps := line[idx+1:]
			if len(deps) > 0 {
				targetDeps = parseDeps(deps)
			}
			continue
		}

		if inTarget && (line[0] == '\t' || line[0] == ' ') {
			cmd := trimBytes(line)
			if len(cmd) > 0 {
				targetCmds = append(targetCmds, string(cmd))
			}
			continue
		}

		if inTarget && len(targetCmds) > 0 {
			rc.processTarget(targetName, targetDeps, targetCmds)
			inTarget = false
			targetName = ""
			targetDeps = nil
			targetCmds = nil
		}
	}

	if inTarget && len(targetCmds) > 0 {
		rc.processTarget(targetName, targetDeps, targetCmds)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return rc.toConfig(), nil
}

func (rc *ReverseConfig) parseVariable(key, val []byte) {
	v := expandVars(string(val))

	switch {
	case bytes.Equal(key, []byte("CC")):
	case bytes.Equal(key, []byte("CXX")):
	case bytes.HasPrefix(key, []byte("CFLAGS")):
		rc.parseFlags(v)
	case bytes.HasPrefix(key, []byte("CXXFLAGS")):
		rc.parseFlags(v)
	case bytes.HasPrefix(key, []byte("LDFLAGS")):
		rc.parseFlags(v)
	case bytes.HasPrefix(key, []byte("LIBS")):
		rc.parseLibs(v)
	case bytes.HasPrefix(key, []byte("INCLUDES")):
		rc.parseIncludes(v)
	case bytes.HasPrefix(key, []byte("TARGET")):
		if v != "" {
			rc.Output = v
			rc.Target = v
		}
	case bytes.HasPrefix(key, []byte("OUT")):
		if v != "" {
			rc.Output = v
		}
	case bytes.HasPrefix(key, []byte("SRC")):
		rc.parseSources(v)
	}
}

func (rc *ReverseConfig) parseCompilerFlags(v string) {
	parts := splitFlags(v)
	for _, p := range parts {
		if strings.HasPrefix(p, "-D") {
			rc.parseDefine(p[2:])
		} else if strings.HasPrefix(p, "-I") {
			rc.Include = append(rc.Include, p[2:])
		} else if strings.HasPrefix(p, "-l") {
			rc.Libs = append(rc.Libs, p[2:])
		} else if strings.HasPrefix(p, "-") {
			rc.Flags = append(rc.Flags, p)
		}
	}
}

func (rc *ReverseConfig) parseFlags(v string) {
	parts := splitFlags(v)
	for _, p := range parts {
		if strings.HasPrefix(p, "-l") {
			rc.Libs = append(rc.Libs, p[2:])
		} else if strings.HasPrefix(p, "-I") {
			rc.Include = append(rc.Include, p[2:])
		} else if strings.HasPrefix(p, "-D") {
			rc.parseDefine(p[2:])
		} else {
			rc.Flags = append(rc.Flags, p)
		}
	}
}

func (rc *ReverseConfig) parseDefine(def string) {
	parts := strings.SplitN(def, "=", 2)
	if len(parts) == 2 {
		rc.Defines[parts[0]] = parts[1]
	} else {
		rc.Defines[parts[0]] = "1"
	}
}

func (rc *ReverseConfig) parseLibs(v string) {
	parts := splitFlags(v)
	for _, p := range parts {
		if strings.HasPrefix(p, "-l") {
			rc.Libs = append(rc.Libs, p[2:])
		} else if !strings.HasPrefix(p, "-") {
			rc.Libs = append(rc.Libs, p)
		}
	}
}

func (rc *ReverseConfig) parseIncludes(v string) {
	parts := splitFlags(v)
	for _, p := range parts {
		if strings.HasPrefix(p, "-I") {
			rc.Include = append(rc.Include, p[2:])
		} else if strings.HasPrefix(p, "-isystem") {
			parts2 := splitFlags(p[8:])
			if len(parts2) > 0 {
				rc.Include = append(rc.Include, parts2[0])
			}
		}
	}
}

func (rc *ReverseConfig) parseSources(v string) {
	parts := splitFlags(v)
	for _, p := range parts {
		if strings.HasSuffix(p, ".c") || strings.HasSuffix(p, ".cc") ||
			strings.HasSuffix(p, ".cpp") || strings.HasSuffix(p, ".s") ||
			strings.HasSuffix(p, ".asm") {
			rc.Source = append(rc.Source, p)
		}
	}
}

func (rc *ReverseConfig) processTarget(name string, deps []string, cmds []string) {
	if name == "" || len(cmds) == 0 {
		return
	}

	if rc.Output == "" && !strings.Contains(name, ".") {
		rc.Output = name
	}

	for _, dep := range deps {
		if strings.HasSuffix(dep, ".c") || strings.HasSuffix(dep, ".cc") ||
			strings.HasSuffix(dep, ".cpp") || strings.HasSuffix(dep, ".s") ||
			strings.HasSuffix(dep, ".asm") {
			rc.Source = append(rc.Source, dep)
		}
	}

	for _, cmd := range cmds {
		rc.parseCommand(cmd)
	}
}

func (rc *ReverseConfig) parseCommand(cmd string) {
	parts := splitFlags(cmd)
	for i, p := range parts {
		switch {
		case strings.HasPrefix(p, "-o"):
			if len(p) > 2 {
				rc.Output = p[2:]
			} else if i+1 < len(parts) {
				rc.Output = parts[i+1]
			}
		case strings.HasPrefix(p, "-D"):
			rc.parseDefine(p[2:])
		case strings.HasPrefix(p, "-I"):
			rc.Include = append(rc.Include, p[2:])
		case strings.HasPrefix(p, "-l"):
			rc.Libs = append(rc.Libs, p[2:])
		case strings.HasSuffix(p, ".c") || strings.HasSuffix(p, ".cc") ||
			strings.HasSuffix(p, ".cpp") || strings.HasSuffix(p, ".s") ||
			strings.HasSuffix(p, ".asm"):
			rc.Source = append(rc.Source, p)
		case strings.HasPrefix(p, "-"):
			rc.Flags = append(rc.Flags, p)
		}
	}
}

func (rc *ReverseConfig) toConfig() *config.Config {
	cfg := &config.Config{
		SourceFiles: rc.Source,
		Output:      rc.Output,
		Libs:        rc.Libs,
		Include:     rc.Include,
		Mode:        "auto",
		Profile:     "balanced",
		Toolchain:   "auto",
	}

	if rc.Target != "" {
		cfg.Target = rc.Target
	}

	if rc.Name != "" {
		cfg.Name = rc.Name
	}

	if len(rc.Defines) > 0 {
		for k, v := range rc.Defines {
			cfg.Flags.Cc = append(cfg.Flags.Cc, "-D"+k+"="+v)
		}
	}

	if len(rc.Flags) > 0 {
		cfg.Flags.Cc = append(cfg.Flags.Cc, rc.Flags...)
	}

	if len(rc.Include) > 0 {
		for _, inc := range rc.Include {
			if inc != "" {
				cfg.Flags.Cc = append(cfg.Flags.Cc, "-I"+inc)
			}
		}
	}

	if len(rc.Libs) > 0 {
		for _, lib := range rc.Libs {
			if lib != "" {
				cfg.Flags.Ld = append(cfg.Flags.Ld, "-l"+lib)
			}
		}
	}

	if cfg.Output == "" && len(cfg.SourceFiles) > 0 {
		base := filepath.Base(cfg.SourceFiles[0])
		ext := filepath.Ext(base)
		cfg.Output = strings.TrimSuffix(base, ext)
		if cfg.Output == "" {
			cfg.Output = "a.out"
		}
	}

	return cfg
}

func trimBytes(b []byte) []byte {
	start := 0
	end := len(b)
	for start < end && (b[start] == ' ' || b[start] == '\t' || b[start] == '\r' || b[start] == '\n') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t' || b[end-1] == '\r' || b[end-1] == '\n') {
		end--
	}
	return b[start:end]
}

func parseDeps(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	parts := bytes.Fields(b)
	if len(parts) == 0 {
		return nil
	}
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		s := string(p)
		if s != "" && !strings.HasPrefix(s, "$") {
			result = append(result, s)
		}
	}
	return result
}

func splitFlags(s string) []string {
	if s == "" {
		return nil
	}
	result := make([]string, 0, 16)
	in := false
	var quote byte
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if in {
			if c == quote {
				in = false
			}
			continue
		}
		if c == '"' || c == '\'' {
			in = true
			quote = c
			continue
		}
		if c == ' ' || c == '\t' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func expandVars(s string) string {
	if !strings.Contains(s, "$") {
		return s
	}
	var buf strings.Builder
	buf.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '$' && i+1 < len(s) {
			if s[i+1] == '(' {
				end := strings.Index(s[i+2:], ")")
				if end > 0 {
					i += end + 2
					continue
				}
			} else if s[i+1] == '{' {
				end := strings.Index(s[i+2:], "}")
				if end > 0 {
					i += end + 2
					continue
				}
			} else if (s[i+1] >= 'a' && s[i+1] <= 'z') || (s[i+1] >= 'A' && s[i+1] <= 'Z') {
				end := i + 1
				for end < len(s) && ((s[end] >= 'a' && s[end] <= 'z') || (s[end] >= 'A' && s[end] <= 'Z') || (s[end] >= '0' && s[end] <= '9') || s[end] == '_') {
					end++
				}
				i = end - 1
				continue
			}
		}
		buf.WriteByte(s[i])
	}
	return buf.String()
}

func ReverseCMake(filename string) (*config.Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rc := &ReverseConfig{
		Source:  make([]string, 0, 16),
		Libs:    make([]string, 0, 8),
		Include: make([]string, 0, 8),
		Flags:   make([]string, 0, 16),
		Defines: make(map[string]string),
	}

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 65536)
	scanner.Buffer(buf, 65536)

	for scanner.Scan() {
		line := scanner.Bytes()
		line = trimBytes(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		if bytes.HasPrefix(line, []byte("project(")) {
			content := line[8 : len(line)-1]
			parts := bytes.Split(content, []byte(" "))
			if len(parts) > 0 {
				rc.Name = string(trimBytes(parts[0]))
			}
			continue
		}

		if bytes.HasPrefix(line, []byte("add_executable(")) {
			content := line[15 : len(line)-1]
			parts := bytes.Split(content, []byte(" "))
			for i, p := range parts {
				p = trimBytes(p)
				if i == 0 {
					if rc.Output == "" {
						rc.Output = string(p)
					}
					continue
				}
				if len(p) > 0 {
					s := string(p)
					if strings.HasSuffix(s, ".c") || strings.HasSuffix(s, ".cc") ||
						strings.HasSuffix(s, ".cpp") || strings.HasSuffix(s, ".s") ||
						strings.HasSuffix(s, ".asm") {
						rc.Source = append(rc.Source, s)
					}
				}
			}
			continue
		}

		if bytes.HasPrefix(line, []byte("add_library(")) {
			content := line[12 : len(line)-1]
			parts := bytes.Split(content, []byte(" "))
			for i, p := range parts {
				p = trimBytes(p)
				if i == 0 {
					if rc.Output == "" {
						rc.Output = "lib" + string(p) + ".a"
					}
					continue
				}
				if len(p) > 0 {
					s := string(p)
					if strings.HasSuffix(s, ".c") || strings.HasSuffix(s, ".cc") ||
						strings.HasSuffix(s, ".cpp") || strings.HasSuffix(s, ".s") ||
						strings.HasSuffix(s, ".asm") {
						rc.Source = append(rc.Source, s)
					}
				}
			}
			continue
		}

		if bytes.HasPrefix(line, []byte("target_include_directories(")) {
			content := line[24 : len(line)-1]
			parts := bytes.Split(content, []byte(" "))
			for i, p := range parts {
				p = trimBytes(p)
				if i > 1 && len(p) > 0 {
					s := string(p)
					if !strings.HasPrefix(s, "PRIVATE") && !strings.HasPrefix(s, "PUBLIC") &&
						!strings.HasPrefix(s, "INTERFACE") {
						rc.Include = append(rc.Include, s)
					}
				}
			}
			continue
		}

		if bytes.HasPrefix(line, []byte("target_link_libraries(")) {
			content := line[21 : len(line)-1]
			parts := bytes.Split(content, []byte(" "))
			for i, p := range parts {
				p = trimBytes(p)
				if i > 0 && len(p) > 0 {
					s := string(p)
					if !strings.HasPrefix(s, "PRIVATE") && !strings.HasPrefix(s, "PUBLIC") &&
						!strings.HasPrefix(s, "INTERFACE") && !strings.Contains(s, "::") {
						rc.Libs = append(rc.Libs, s)
					}
				}
			}
			continue
		}

		if bytes.HasPrefix(line, []byte("set(")) {
			content := line[4 : len(line)-1]
			parts := bytes.SplitN(content, []byte(" "), 2)
			if len(parts) == 2 {
				key := string(trimBytes(parts[0]))
				val := string(trimBytes(parts[1]))
				switch {
				case key == "CMAKE_CXX_FLAGS" || key == "CMAKE_C_FLAGS":
					rc.parseFlags(val)
				case key == "CMAKE_EXE_LINKER_FLAGS":
					rc.parseFlags(val)
				}
			}
			continue
		}

		if bytes.HasPrefix(line, []byte("find_package(")) {
			content := line[13 : len(line)-1]
			parts := bytes.Split(content, []byte(" "))
			if len(parts) > 0 {
				pkg := string(trimBytes(parts[0]))
				rc.Libs = append(rc.Libs, pkg)
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return rc.toConfig(), nil
}

func ReverseFile(filename string) (*config.Config, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mk", ".makefile":
		return ReverseMakefile(filename)
	case ".cmake", ".txt":
		if strings.Contains(strings.ToLower(filename), "cmake") {
			return ReverseCMake(filename)
		}
		fallthrough
	default:
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			return nil, errors.New("empty file")
		}
		if bytes.Contains(data, []byte("add_executable")) || bytes.Contains(data, []byte("project(")) {
			return ReverseCMake(filename)
		}
		if bytes.Contains(data, []byte("CC=")) || bytes.Contains(data, []byte("CC =")) ||
			bytes.Contains(data, []byte("TARGET=")) || bytes.Contains(data, []byte("TARGET =")) ||
			bytes.Contains(data, []byte("SRCS=")) || bytes.Contains(data, []byte("SRCS =")) {
			return ReverseMakefile(filename)
		}
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) > 0 && (line[0] == '\t' || line[0] == ' ') {
				return ReverseMakefile(filename)
			}
			if idx := bytes.IndexByte(line, ':'); idx > 0 {
				before := trimBytes(line[:idx])
				if len(before) > 0 && !bytes.Contains(before, []byte(" ")) {
					return ReverseMakefile(filename)
				}
			}
		}
		return nil, errors.New("unsupported build file format")
	}
}
