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
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func parseMakefileVars(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	vars := make(map[string]string)
	scanner := bufio.NewScanner(f)
	var curName string
	var curOp string
	var curVal strings.Builder
	re := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*([:+]?=)\s*(.*)$`)
	for scanner.Scan() {
		line := scanner.Text()
		if curName != "" {
			if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ") {
				trimmed := strings.TrimSpace(line)
				if strings.HasSuffix(trimmed, "\\") {
					trimmed = strings.TrimSuffix(trimmed, "\\")
				}
				if curVal.Len() > 0 {
					curVal.WriteByte(' ')
				}
				curVal.WriteString(trimmed)
				continue
			}
			val := strings.TrimSpace(curVal.String())
			if curOp == "+=" {
				if prev, ok := vars[curName]; ok && prev != "" {
					vars[curName] = prev + " " + val
				} else {
					vars[curName] = val
				}
			} else {
				vars[curName] = val
			}
			curName = ""
			curOp = ""
			curVal.Reset()
		}

		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		curName = m[1]
		curOp = m[2]
		v := strings.TrimSpace(m[3])
		if strings.HasSuffix(v, "\\") {
			v = strings.TrimSuffix(v, "\\")
		}
		curVal.WriteString(v)
	}
	if curName != "" {
		val := strings.TrimSpace(curVal.String())
		if curOp == "+=" {
			if prev, ok := vars[curName]; ok && prev != "" {
				vars[curName] = prev + " " + val
			} else {
				vars[curName] = val
			}
		} else {
			vars[curName] = val
		}
	}
	return vars, scanner.Err()
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
