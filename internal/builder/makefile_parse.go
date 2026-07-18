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

package builder

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unsafe"
)

func parseMakefileVars(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	vars := make(map[string]string)

	var curName []byte
	var curOp []byte
	var curVal []byte
	cont := false

	nextLine := func(start int) (line []byte, next int) {
		if start >= len(data) {
			return nil, len(data)
		}
		end := start
		for end < len(data) && data[end] != '\n' {
			end++
		}
		line = data[start:end]
		if end < len(data) && data[end] == '\n' {
			next = end + 1
		} else {
			next = end
		}
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		return
	}

	start := 0
	for start < len(data) {
		line, nstart := nextLine(start)
		start = nstart
		if len(curName) > 0 && len(line) > 0 && (line[0] == '\t' || line[0] == ' ') {
			t := trimSpaceBytes(line)
			if len(t) > 0 && t[len(t)-1] == '\\' {
				t = t[:len(t)-1]
				cont = true
			} else {
				cont = false
			}
			if len(curVal) > 0 {
				curVal = append(curVal, ' ')
			}
			curVal = append(curVal, t...)
			if cont {
				continue
			}
			key := bytesToString(trimSpaceBytes(curName))
			val := bytesToString(trimSpaceBytes(curVal))
			if bytes.Equal(curOp, []byte("+=")) {
				if prev, ok := vars[key]; ok && prev != "" {
					vars[key] = prev + " " + val
				} else {
					vars[key] = val
				}
			} else {
				vars[key] = val
			}
			curName = nil
			curOp = nil
			curVal = nil
			continue
		}

		eq := bytes.IndexByte(line, '=')
		if eq == -1 {
			continue
		}
		i := eq - 1
		for i >= 0 && (line[i] == ' ' || line[i] == '\t') {
			i--
		}
		var op []byte
		if i >= 0 && (line[i] == ':' || line[i] == '+') {
			op = []byte{line[i], '='}
			j := i - 1
			for j >= 0 && (line[j] == ' ' || line[j] == '\t') {
				j--
			}
			name := line[:j+1]
			name = trimSpaceBytes(name)
			curName = append(curName[:0], name...)
		} else {
			op = []byte{'='}
			name := line[:i+1]
			name = trimSpaceBytes(name)
			curName = append(curName[:0], name...)
		}
		curOp = append(curOp[:0], op...)
		v := line[eq+1:]
		v = trimSpaceBytes(v)
		if len(v) > 0 && v[len(v)-1] == '\\' {
			v = v[:len(v)-1]
			cont = true
		} else {
			cont = false
		}
		curVal = append(curVal[:0], v...)
		if cont {
			continue
		}
		key := bytesToString(trimSpaceBytes(curName))
		val := bytesToString(trimSpaceBytes(curVal))
		if bytes.Equal(curOp, []byte("+=")) {
			if prev, ok := vars[key]; ok && prev != "" {
				vars[key] = prev + " " + val
			} else {
				vars[key] = val
			}
		} else {
			vars[key] = val
		}
		curName = nil
		curOp = nil
		curVal = nil
	}

	return vars, nil
}

func trimSpaceBytes(b []byte) []byte {
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

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func makefileCandidates(rootDir string) []string {
	candidates := []string{
		filepath.Join(rootDir, "objs", "Makefile"),
		filepath.Join(rootDir, "Makefile"),
	}
	autoDir := filepath.Join(rootDir, "auto")
	if entries, err := os.ReadDir(autoDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			candidates = append(candidates, filepath.Join(autoDir, entry.Name()))
		}
	}
	return candidates
}

func normalizeMakefileToken(rootDir, token string) string {
	token = strings.TrimSpace(token)
	token = strings.Trim(token, `"'`)
	if token == "" {
		return ""
	}
	if filepath.IsAbs(token) {
		return token
	}
	return filepath.Join(rootDir, token)
}

func addMakefileSource(rootDir, token string, sources map[string]struct{}) {
	if strings.Contains(token, "$(") || strings.Contains(token, "${") {
		return
	}
	token = strings.TrimSpace(strings.Trim(token, `"'`))
	if token == "" {
		return
	}
	ext := strings.ToLower(filepath.Ext(token))
	if !supportedSourceInclude(ext) && ext != ".o" {
		return
	}
	if ext == ".o" {
		inferSourcesFromObjectTarget(rootDir, token, sources)
		return
	}
	path := normalizeMakefileToken(rootDir, token)
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		sources[path] = struct{}{}
	}
}

func inferSourcesFromObjectTarget(rootDir, obj string, sources map[string]struct{}) {
	obj = strings.TrimSpace(strings.Trim(obj, `"'`))
	if strings.HasPrefix(obj, "./") {
		obj = strings.TrimPrefix(obj, "./")
	}
	if strings.HasPrefix(obj, "objs/") {
		obj = strings.TrimPrefix(obj, "objs/")
	}
	obj = strings.TrimSuffix(obj, ".o")
	if obj == "" {
		return
	}
	extensions := []string{".c", ".cc", ".cpp", ".cxx", ".m", ".mm", ".S", ".s", ".asm", ".fasm"}
	for _, ext := range extensions {
		path := filepath.Join(rootDir, obj+ext)
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			sources[path] = struct{}{}
			return
		}
	}
}

func collectSourcesFromMakefileValues(rootDir, value string, sources map[string]struct{}) {
	for _, token := range strings.Fields(value) {
		if strings.HasPrefix(token, "-I") || strings.HasPrefix(token, "-D") || strings.HasPrefix(token, "-L") || strings.HasPrefix(token, "-W") {
			continue
		}
		addMakefileSource(rootDir, token, sources)
	}
}

func collectSourcesFromObjectTargets(rootDir, content string, sources map[string]struct{}) {
	re := regexp.MustCompile(`(?m)^([^\s:]+\.o)\s*:`)
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		addMakefileSource(rootDir, m[1], sources)
	}
}

func discoverMakefileSettings(rootDir string) (includes []string, cflags string, ldflags string) {
	candidates := makefileCandidates(rootDir)
	for _, p := range candidates {
		if info, err := os.Stat(p); err != nil || info.IsDir() {
			continue
		}
		vars, err := parseMakefileVars(p)
		if err != nil {
			continue
		}
		if v, ok := vars["CPPFLAGS"]; ok {
			cflags = joinFlags(cflags, v)
		}
		if v, ok := vars["CFLAGS"]; ok {
			cflags = joinFlags(cflags, v)
		}
		if v, ok := vars["LDFLAGS"]; ok {
			ldflags = joinFlags(ldflags, v)
		}
		if v, ok := vars["LDLIBS"]; ok {
			ldflags = joinFlags(ldflags, v)
		}
		for _, val := range vars {
			fields := strings.Fields(val)
			for i := 0; i < len(fields); i++ {
				f := fields[i]
				if strings.HasPrefix(f, "-I") {
					p := strings.TrimSpace(strings.TrimPrefix(f, "-I"))
					if p == "" {
						if i+1 < len(fields) {
							i++
							p = fields[i]
						} else {
							continue
						}
					}
					if !filepath.IsAbs(p) {
						p = filepath.Join(rootDir, p)
					}
					if a, err := filepath.Abs(p); err == nil {
						includes = append(includes, a)
					} else {
						includes = append(includes, p)
					}
				}
			}
		}
	}
	return includes, strings.TrimSpace(cflags), strings.TrimSpace(ldflags)
}

func discoverMakefileSources(rootDir string) ([]string, error) {
	sources := make(map[string]struct{})
	candidates := makefileCandidates(rootDir)
	for _, p := range candidates {
		if info, err := os.Stat(p); err != nil || info.IsDir() {
			continue
		}
		content, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		vars, err := parseMakefileVars(p)
		if err == nil {
			for _, v := range vars {
				collectSourcesFromMakefileValues(rootDir, v, sources)
			}
		}
		collectSourcesFromObjectTargets(rootDir, string(content), sources)
	}
	if len(sources) == 0 {
		return nil, nil
	}
	result := make([]string, 0, len(sources))
	for src := range sources {
		result = append(result, src)
	}
	sort.Strings(result)
	return result, nil
}

func joinFlags(a, b string) string {
	if strings.TrimSpace(a) == "" {
		return strings.TrimSpace(b)
	}
	if strings.TrimSpace(b) == "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(a + " " + b)
}
