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
		if _, err := exec.LookPath("nm"); err != nil {
			detectedTool = "nm"
		} else if _, err := exec.LookPath("objdump"); err != nil {
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

	sem := make(chan struct{}, 16)
	resultsChan := make(chan result, len(objFiles))
	var wg sync.WaitGroup

	for _, obj := range objFiles {
		wg.Add(1)
		go func(objFile string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				resultsChan <- result{obj: objFile, err: ctx.Err()}
				return
			}

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
	return parseNmOutput(objPath, string(out)), nil
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
	return parseObjdumpOutput(objPath, string(out)), nil
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
	return parseReadelfOutput(objPath, string(out)), nil
}

func parseNmOutput(objPath, text string) []SymbolInfo {
	var syms []SymbolInfo
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		typ := fields[1]
		if typ != "T" && typ != "D" && typ != "B" {
			continue
		}
		name := fields[2]
		if name == "" || name == "_start" || strings.HasPrefix(name, ".") {
			continue
		}
		syms = append(syms, SymbolInfo{File: objPath, Name: name, Type: "global"})
	}
	return syms
}

func parseObjdumpOutput(objPath, text string) []SymbolInfo {
	var syms []SymbolInfo
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "g") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		section := fields[2]
		if section == "UND" || section == "*ABS*" {
			continue
		}
		name := fields[len(fields)-1]
		if name == "" || name == "_start" || strings.HasPrefix(name, ".") {
			continue
		}
		syms = append(syms, SymbolInfo{File: objPath, Name: name, Type: "global"})
	}
	return syms
}

func parseReadelfOutput(objPath, text string) []SymbolInfo {
	var syms []SymbolInfo
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "GLOBAL") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		sectionIdx := 6
		if sectionIdx < len(fields) && fields[sectionIdx] == "UND" {
			continue
		}
		name := fields[len(fields)-1]
		if name == "" || name == "_start" || strings.HasPrefix(name, ".") {
			continue
		}
		syms = append(syms, SymbolInfo{File: objPath, Name: name, Type: "global"})
	}
	return syms
}
