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

package linker

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/forgezero-cli/ForgeZero/internal/drivers/concurrency"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

type SymbolInfo struct {
	File  string
	Name  string
	Type  string
	Size  int
	Bound string
}

var (
	detectedTool     string
	detectedToolOnce sync.Once
)

func getSymbolsTool() string {
	detectedToolOnce.Do(func() {
		if _, err := exec.LookPath("nm"); err == nil {
			detectedTool = "nm"
		} else if _, err := exec.LookPath("objdump"); err == nil {
			detectedTool = "objdump"
		} else {
			detectedTool = "readelf"
		}
	})
	return detectedTool
}

func CheckDuplicateSymbols(ctx context.Context, objFiles []string, verbose bool) error {
	if len(objFiles) <= 1 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	type result struct {
		obj  string
		syms []SymbolInfo
		err  error
	}

	sem := concurrency.NewSemaphore(16)
	resultsChan := make(chan result, len(objFiles))
	var wg concurrency.WaitGroup

	for _, obj := range objFiles {
		wg.Add(1)
		go func(objFile string) {
			defer wg.Done()
			if err := sem.AcquireContext(ctx, 1); err != nil {
				resultsChan <- result{obj: objFile, err: err}
				return
			}
			defer sem.Release(1)

			if err := utils.CheckFileExists(objFile); err != nil {
				resultsChan <- result{obj: objFile, err: err}
				return
			}

			syms, err := readSymbols(ctx, objFile, verbose)
			resultsChan <- result{obj: objFile, syms: syms, err: err}
		}(obj)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	symbolMap := make(map[string][]SymbolInfo)
	for res := range resultsChan {
		if res.err != nil {
			if verbose {
				var b strings.Builder
				b.WriteString("Warning: cannot read symbols from ")
				b.WriteString(res.obj)
				b.WriteString(": ")
				b.WriteString(res.err.Error())
				b.WriteByte('\n')
				os.Stderr.WriteString(b.String())
			}
			continue
		}
		for _, sym := range res.syms {
			symbolMap[sym.Name] = append(symbolMap[sym.Name], sym)
		}
	}

	var dupBuf strings.Builder
	first := true
	for name, syms := range symbolMap {
		if len(syms) > 1 && shouldCheckDuplicate(name) {
			if first {
				first = false
			} else {
				dupBuf.WriteByte('\n')
			}
			dupBuf.WriteString("  symbol '")
			dupBuf.WriteString(name)
			dupBuf.WriteString("' defined in:\n    ")
			for i, s := range syms {
				if i > 0 {
					dupBuf.WriteString("\n    ")
				}
				dupBuf.WriteString(s.File)
				dupBuf.WriteString(" (")
				dupBuf.WriteString(s.Type)
				dupBuf.WriteByte(')')
			}
		}
	}

	if dupBuf.Len() > 0 {
		var errBuf strings.Builder
		errBuf.WriteString("duplicate global symbols found:\n")
		errBuf.WriteString(dupBuf.String())
		errBuf.WriteString("\nUse -no-symbol-check to skip this check")
		return errors.New(errBuf.String())
	}
	return nil
}

func shouldCheckDuplicate(name string) bool {
	if name == "" || name == "_end" || name == "_edata" || name == "__bss_start" {
		return false
	}
	if strings.HasPrefix(name, ".L") || strings.HasPrefix(name, "debug_") {
		return false
	}
	return true
}

func readSymbols(ctx context.Context, objPath string, verbose bool) ([]SymbolInfo, error) {
	hash, err := utils.HashFile(objPath)
	var cacheFile string
	if err == nil {
		if cacheDir, cerr := os.UserCacheDir(); cerr == nil {
			cacheFile = filepath.Join(cacheDir, "fzt", "symbols", hash[:2], hash+".syms")

			if data, rerr := os.ReadFile(cacheFile); rerr == nil {
				if verbose {
					var b strings.Builder
					b.WriteString("Symbol cache hit for ")
					b.WriteString(objPath)
					b.WriteByte('\n')
					os.Stderr.WriteString(b.String())
				}
				return deserializeSymbols(data, objPath), nil
			}
		}
	}

	var syms []SymbolInfo
	tool := getSymbolsTool()
	switch tool {
	case "nm":
		syms, err = readSymbolsWithNm(ctx, objPath, verbose)
	case "objdump":
		syms, err = readSymbolsWithObjdump(ctx, objPath, verbose)
	default:
		syms, err = readSymbolsWithReadelf(ctx, objPath, verbose)
	}

	if err != nil {
		return nil, err
	}

	if cacheFile != "" {
		_ = os.MkdirAll(filepath.Dir(cacheFile), 0o755)
		_ = os.WriteFile(cacheFile, serializeSymbols(syms), 0o644)
	}

	return syms, nil
}

func serializeSymbols(syms []SymbolInfo) []byte {
	var buf bytes.Buffer
	for _, s := range syms {
		buf.WriteString(s.Name)
		buf.WriteByte('\t')
		buf.WriteString(s.Type)
		buf.WriteByte('\t')
		buf.WriteString(strconv.Itoa(s.Size))
		buf.WriteByte('\t')
		buf.WriteString(s.Bound)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func deserializeSymbols(data []byte, objPath string) []SymbolInfo {
	var syms []SymbolInfo
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}
		size, _ := strconv.Atoi(parts[2])
		syms = append(syms, SymbolInfo{
			File:  objPath,
			Name:  parts[0],
			Type:  parts[1],
			Size:  size,
			Bound: parts[3],
		})
	}
	return syms
}

func readSymbolsWithNm(ctx context.Context, objPath string, verbose bool) ([]SymbolInfo, error) {
	out, err := utils.RunCommandOutput(ctx, "nm", "-g", objPath)
	if err != nil {
		var b strings.Builder
		b.WriteString("nm ")
		b.WriteString(objPath)
		b.WriteString(": ")
		b.WriteString(err.Error())
		return nil, errors.New(b.String())
	}
	return parseNmOutputBytes(objPath, out), nil
}

func readSymbolsWithObjdump(ctx context.Context, objPath string, verbose bool) ([]SymbolInfo, error) {
	out, err := utils.RunCommandOutput(ctx, "objdump", "-t", objPath)
	if err != nil {
		var b strings.Builder
		b.WriteString("objdump ")
		b.WriteString(objPath)
		b.WriteString(": ")
		b.WriteString(err.Error())
		return nil, errors.New(b.String())
	}
	return parseObjdumpOutputBytes(objPath, out), nil
}

func readSymbolsWithReadelf(ctx context.Context, objPath string, verbose bool) ([]SymbolInfo, error) {
	out, err := utils.RunCommandOutput(ctx, "readelf", "-s", objPath)
	if err != nil {
		var b strings.Builder
		b.WriteString("readelf ")
		b.WriteString(objPath)
		b.WriteString(": ")
		b.WriteString(err.Error())
		return nil, errors.New(b.String())
	}
	return parseReadelfOutputBytes(objPath, out), nil
}

func parseNmOutput(objPath, text string) []SymbolInfo {
	return parseNmOutputBytes(objPath, []byte(text))
}

func parseNmOutputBytes(objPath string, data []byte) []SymbolInfo {
	syms := make([]SymbolInfo, 0, 64)
	var i int
	for i < len(data) {
		j := i
		for j < len(data) && data[j] != '\n' {
			j++
		}
		line := data[i:j]
		i = j + 1

		if len(line) == 0 {
			continue
		}

		p0 := 0
		for p0 < len(line) && (line[p0] == ' ' || line[p0] == '\t' || line[p0] == '\r') {
			p0++
		}
		p1 := p0
		for p1 < len(line) && line[p1] != ' ' && line[p1] != '\t' && line[p1] != '\r' {
			p1++
		}
		p2 := p1
		for p2 < len(line) && (line[p2] == ' ' || line[p2] == '\t' || line[p2] == '\r') {
			p2++
		}
		p3 := p2
		for p3 < len(line) && line[p3] != ' ' && line[p3] != '\t' && line[p3] != '\r' {
			p3++
		}
		p4 := p3
		for p4 < len(line) && (line[p4] == ' ' || line[p4] == '\t' || line[p4] == '\r') {
			p4++
		}
		p5 := p4
		for p5 < len(line) && line[p5] != ' ' && line[p5] != '\t' && line[p5] != '\r' {
			p5++
		}

		if p5-p4 <= 0 {
			continue
		}

		typStart := p2
		typLen := p3 - p2
		if typLen == 0 {
			continue
		}

		var typ byte
		if typLen > 0 {
			typ = line[typStart]
		}

		if !(typ == 'T' || typ == 'D' || typ == 'B') {
			continue
		}

		name := string(line[p4:p5])
		if name == "" || name == "_start" || (len(name) > 0 && name[0] == '.') {
			continue
		}

		syms = append(syms, SymbolInfo{File: objPath, Name: name, Type: "global"})
	}
	return syms
}

func parseObjdumpOutput(objPath, text string) []SymbolInfo {
	return parseObjdumpOutputBytes(objPath, []byte(text))
}

func parseObjdumpOutputBytes(objPath string, data []byte) []SymbolInfo {
	syms := make([]SymbolInfo, 0, 64)
	var i int
	for i < len(data) {
		j := i
		for j < len(data) && data[j] != '\n' {
			j++
		}
		line := data[i:j]
		i = j + 1
		if len(line) == 0 {
			continue
		}

		if !bytes.Contains(line, []byte{'g'}) {
			continue
		}

		// tokenization: fields separated by space/tab
		p := 0
		idx := 0
		var f2 []byte
		var last []byte

		for p < len(line) {
			for p < len(line) && (line[p] == ' ' || line[p] == '\t' || line[p] == '\r') {
				p++
			}
			if p >= len(line) {
				break
			}
			start := p
			for p < len(line) && line[p] != ' ' && line[p] != '\t' && line[p] != '\r' {
				p++
			}
			end := p
			if end <= start {
				break
			}
			field := line[start:end]
			if idx == 2 {
				f2 = field
			}
			last = field
			idx++
		}

		if idx < 6 {
			continue
		}
		if f2 == nil {
			continue
		}
		if string(f2) == "UND" || string(f2) == "*ABS*" {
			continue
		}

		name := last
		if len(name) == 0 {
			continue
		}
		if bytes.Equal(name, []byte("_start")) {
			continue
		}
		if name[0] == '.' {
			continue
		}

		syms = append(syms, SymbolInfo{File: objPath, Name: string(name), Type: "global"})
	}
	return syms
}

func parseReadelfOutput(objPath, text string) []SymbolInfo {
	return parseReadelfOutputBytes(objPath, []byte(text))
}

func parseReadelfOutputBytes(objPath string, data []byte) []SymbolInfo {
	syms := make([]SymbolInfo, 0, 64)
	var i int
	for i < len(data) {
		j := i
		for j < len(data) && data[j] != '\n' {
			j++
		}
		line := data[i:j]
		i = j + 1
		if len(line) == 0 {
			continue
		}

		if !bytes.Contains(line, []byte("GLOBAL")) {
			continue
		}

		p := 0
		idx := 0
		var field6 []byte
		var last []byte
		for p < len(line) {
			for p < len(line) && (line[p] == ' ' || line[p] == '\t' || line[p] == '\r') {
				p++
			}
			if p >= len(line) {
				break
			}
			start := p
			for p < len(line) && line[p] != ' ' && line[p] != '\t' && line[p] != '\r' {
				p++
			}
			end := p
			if end <= start {
				break
			}
			field := line[start:end]
			if idx == 6 {
				field6 = field
			}
			last = field
			idx++
		}

		if idx < 8 {
			continue
		}
		if field6 != nil && bytes.Equal(field6, []byte("UND")) {
			continue
		}

		name := last
		if len(name) == 0 {
			continue
		}
		if bytes.Equal(name, []byte("_start")) {
			continue
		}
		if name[0] == '.' {
			continue
		}

		syms = append(syms, SymbolInfo{File: objPath, Name: string(name), Type: "global"})
	}
	return syms
}
