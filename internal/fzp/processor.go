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

package fzp

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	fzerr "github.com/forgezero-cli/ForgeZero/internal/errors"
	"github.com/forgezero-cli/ForgeZero/internal/logger"
)

type Options struct {
	RootDir     string
	AllowedDirs []string
	Macros      map[string]string
}

type Definition struct {
	Name  string
	Value string
}

type condState struct {
	active       bool
	branchTaken  bool
	parentActive bool
}

type Processor struct {
	rootDir      string
	allowedDirs  map[string]struct{}
	macros       map[string]string
	cache        map[string]string
	includeStack []string
	condStack    []condState
}

func NewProcessor(opts Options) *Processor {
	root := opts.RootDir
	if root == "" {
		root = "."
	}
	allowed := make(map[string]struct{}, len(opts.AllowedDirs)+1)
	allowed[absPath(root)] = struct{}{}
	for _, dir := range opts.AllowedDirs {
		if dir == "" {
			continue
		}
		allowed[absPath(dir)] = struct{}{}
	}
	macros := map[string]string{}
	for k, v := range opts.Macros {
		macros[k] = v
	}
	return &Processor{rootDir: root, allowedDirs: allowed, macros: macros, cache: map[string]string{}}
}

func (p *Processor) Process(path string, opts Options) (string, error) {
	if path == "" {
		return "", nil
	}
	resolved := path
	if filepath.IsAbs(resolved) {
		if !p.isAllowedPath(resolved) {
			return "", nil
		}
	} else {
		resolved = filepath.Join(p.rootDir, resolved)
	}
	resolved = filepath.Clean(resolved)
	if p.isInStack(resolved) {
		return "", fzerr.NewMsg(fzerr.CodeIncludeFailed, "include cycle detected: "+resolved)
	}
	cacheKey := p.cacheKey(resolved, opts.Macros)
	if cached, ok := p.cache[cacheKey]; ok {
		logger.Debug("f zp cache hit: " + resolved + "\n")
		return cached, nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", fzerr.NewMsg(fzerr.CodeFileNotFound, err.Error())
	}
	p.includeStack = append(p.includeStack, resolved)
	prevCond := append([]condState(nil), p.condStack...)
	p.condStack = p.condStack[:0]
	defer func() {
		if len(p.includeStack) > 0 {
			p.includeStack = p.includeStack[:len(p.includeStack)-1]
		}
		p.condStack = prevCond
	}()
	lines := strings.Split(string(data), "\n")
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			result, keep, err := p.handleDirective(trimmed, resolved)
			if err != nil {
				return "", err
			}
			if keep && result != "" {
				out = append(out, result)
			}
			continue
		}
		if p.shouldEmit() && strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	processed := strings.Join(out, "\n")
	p.cache[cacheKey] = processed
	return processed, nil
}

func (p *Processor) ParseDefinitions(output string) (map[string]string, error) {
	defs := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#define ") {
			parts := strings.Fields(trimmed)
			if len(parts) < 2 {
				continue
			}
			name := parts[1]
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "#define "+name))
			defs[name] = value
		}
	}
	return defs, nil
}

func (p *Processor) ConvertToConfig(defs map[string]string) (map[string]string, error) {
	out := map[string]string{}
	for k, v := range defs {
		out[k] = v
	}
	return out, nil
}

func (p *Processor) expandMacros(value string) string {
	if len(p.macros) == 0 {
		return value 
	}

	parts := strings.Fields(value) 
	if len(parts) == 1 && strings.HasPrefix(parts[0], "OUTPUT") {
		if val, ok := p.macros[parts[0]]; ok {
			return val 
	}
	}
	for name, replacment := range p.macros {
		value = strings.ReplaceAll(value, name, replacment)
	}

	return value 
}

func (p *Processor) handleDirective(line, currentPath string) (string, bool, error) {
	if !strings.HasPrefix(line, "#") {
		return line, true, nil
	}
	if strings.HasPrefix(line, "#define ") {
		if !p.shouldEmit() {
			return "", false, nil
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := parts[1]
			value := strings.TrimSpace(strings.TrimPrefix(line, "#define "+name))
			p.macros[name] = p.expandMacros(value)
		}
		return "", false, nil
	}
	if strings.HasPrefix(line, "#undef ") {
		if !p.shouldEmit() {
			return "", false, nil
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			delete(p.macros, parts[1])
		}
		return "", false, nil
	}
	if strings.HasPrefix(line, "#include") {
		if !p.shouldEmit() {
			return "", false, nil
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			includePath := strings.Trim(parts[1], "\"")
			includePath = strings.Trim(includePath, "<>")
			if includePath == "" {
				return "", false, nil
			}
			resolved, err := p.resolveIncludePath(includePath, currentPath)
			if err != nil {
				return "", false, err
			}
			included, err := p.Process(resolved, Options{RootDir: filepath.Dir(resolved), Macros: p.macros})
			if err != nil {
				return "", false, err
			}
			if included != "" {
				return included, false, nil
			}
		}
		return "", false, nil
	}
	if strings.HasPrefix(line, "#ifdef") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			parentActive := len(p.condStack) == 0 || p.condStack[len(p.condStack)-1].active
			_, ok := p.macros[parts[1]]
			p.condStack = append(p.condStack, condState{active: parentActive && ok, branchTaken: parentActive && ok, parentActive: parentActive})
			return "", false, nil
		}
		return "", false, nil
	}
	if strings.HasPrefix(line, "#ifndef") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			parentActive := len(p.condStack) == 0 || p.condStack[len(p.condStack)-1].active
			_, ok := p.macros[parts[1]]
			p.condStack = append(p.condStack, condState{active: parentActive && !ok, branchTaken: parentActive && !ok, parentActive: parentActive})
			return "", false, nil
		}
		return "", false, nil
	}
	if strings.HasPrefix(line, "#if") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			parentActive := len(p.condStack) == 0 || p.condStack[len(p.condStack)-1].active
			parser := newParser(strings.Join(parts[1:], " "), makeMacroMap(p.macros))
			value := parser.parse() != 0
			p.condStack = append(p.condStack, condState{active: parentActive && value, branchTaken: parentActive && value, parentActive: parentActive})
			return "", false, nil
		}
		return "", false, nil
	}
	if strings.HasPrefix(line, "#elif") {
		if len(p.condStack) == 0 {
			return "", false, nil
		}
		state := &p.condStack[len(p.condStack)-1]
		if state.parentActive && !state.branchTaken {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				parser := newParser(strings.Join(parts[1:], " "), makeMacroMap(p.macros))
				state.active = parser.parse() != 0
				state.branchTaken = state.active
			}
		} else {
			state.active = false
		}
		return "", false, nil
	}
	if strings.HasPrefix(line, "#else") {
		if len(p.condStack) == 0 {
			return "", false, nil
		}
		state := &p.condStack[len(p.condStack)-1]
		if state.parentActive && !state.branchTaken {
			state.active = true
			state.branchTaken = true
		} else {
			state.active = false
		}
		return "", false, nil
	}
	if strings.HasPrefix(line, "#endif") {
		if len(p.condStack) > 0 {
			p.condStack = p.condStack[:len(p.condStack)-1]
		}
		return "", false, nil
	}
	if strings.HasPrefix(line, "#error") {
		return "", false, fzerr.NewMsg(fzerr.CodePreprocessFailed, line)
	}
	if strings.HasPrefix(line, "#pragma") {
		return "", false, nil
	}
	return line, true, nil
}

func (p *Processor) shouldEmit() bool {
	if len(p.condStack) == 0 {
		return true
	}
	state := p.condStack[len(p.condStack)-1]
	return state.parentActive && state.active
}

func (p *Processor) resolveIncludePath(includePath, currentPath string) (string, error) {
	if includePath == "" {
		return "", nil
	}
	if filepath.IsAbs(includePath) {
		if p.isAllowedPath(includePath) {
			return filepath.Clean(includePath), nil
		}
		return "", fzerr.NewMsg(fzerr.CodeIncludeFailed, "include outside allowed paths: "+includePath)
	}
	candidates := []string{filepath.Join(filepath.Dir(currentPath), includePath)}
	for dir := range p.allowedDirs {
		candidates = append(candidates, filepath.Join(dir, includePath))
	}
	for _, candidate := range candidates {
		if p.isAllowedPath(candidate) {
			if _, err := os.Stat(candidate); err == nil {
				return filepath.Clean(candidate), nil
			}
		}
	}
	return "", fzerr.NewMsg(fzerr.CodeIncludeFailed, "cannot resolve include: "+includePath)
}

func (p *Processor) isInStack(path string) bool {
	for _, item := range p.includeStack {
		if item == path {
			return true
		}
	}
	return false
}

func (p *Processor) cacheKey(path string, macros map[string]string) string {
	sum := sha256.New()
	_, _ = sum.Write([]byte(path))
	if len(macros) > 0 {
		keys := make([]string, 0, len(macros))
		for k := range macros {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			_, _ = sum.Write([]byte{0})
			_, _ = sum.Write([]byte(k))
			_, _ = sum.Write([]byte(macros[k]))
		}
	}
	for _, env := range os.Environ() {
		_, _ = sum.Write([]byte{0})
		_, _ = sum.Write([]byte(env))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func (p *Processor) isAllowedPath(path string) bool {
	abs := absPath(path)
	_, ok := p.allowedDirs[abs]
	if ok {
		return true
	}
	for dir := range p.allowedDirs {
		if strings.HasPrefix(abs, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func absPath(path string) string {
	cleaned := filepath.Clean(path)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return cleaned
	}
	return abs
}

func Process(path string, opts Options) (string, error) {
	return NewProcessor(opts).Process(path, opts)
}

func ParseDefinitions(output string) (map[string]string, error) {
	return NewProcessor(Options{}).ParseDefinitions(output)
}

func ConvertToConfig(defs map[string]string) (map[string]string, error) {
	return NewProcessor(Options{}).ConvertToConfig(defs)
}

func Example() string {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	for _, name := range sortedKeys(map[string]string{"OUTPUT": "app", "MODE": "raw"}) {
		_, _ = w.WriteString(name)
		_, _ = w.WriteString("=")
		_, _ = w.WriteString("\"")
		_, _ = w.WriteString("value")
		_, _ = w.WriteString("\"\n")
	}
	_ = w.Flush()
	return buf.String()
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	for _, k := range keys {
		_ = k
	}
	return keys
}

func FormatDefinitions(defs map[string]string) string {
	keys := make([]string, 0, len(defs))
	for k := range defs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(strconv.Quote(defs[k]))
		b.WriteString("\n")
	}
	return b.String()
}
