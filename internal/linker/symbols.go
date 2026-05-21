package linker

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"fz/internal/utils"
)

type SymbolInfo struct {
	File  string
	Name  string
	Type  string
	Size  int
	Bound string
}

func CheckDuplicateSymbols(ctx context.Context, objFiles []string, verbose bool) error {
	if len(objFiles) <= 1 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	symbolMap := make(map[string][]SymbolInfo)
	for _, obj := range objFiles {
		if err := utils.CheckFileExists(obj); err != nil {
			return err
		}
		syms, err := readSymbols(ctx, obj, verbose)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: cannot read symbols from %s: %v\n", obj, err)
			}
			continue
		}
		for _, sym := range syms {
			symbolMap[sym.Name] = append(symbolMap[sym.Name], sym)
		}
	}
	duplicates := []string{}
	for name, syms := range symbolMap {
		if len(syms) > 1 && shouldCheckDuplicate(name) {
			dup := fmt.Sprintf("symbol '%s' defined in:", name)
			for _, s := range syms {
				dup += fmt.Sprintf(" %s", s.File)
			}
			duplicates = append(duplicates, dup)
		}
	}
	if len(duplicates) > 0 {
		return fmt.Errorf("duplicate global symbols found:\n%s\nUse -no-symbol-check to skip this check", strings.Join(duplicates, "\n"))
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
	if _, err := exec.LookPath("nm"); err == nil {
		return readSymbolsWithNm(ctx, objPath, verbose)
	}
	if _, err := exec.LookPath("objdump"); err == nil {
		return readSymbolsWithObjdump(ctx, objPath, verbose)
	}
	return readSymbolsWithReadelf(ctx, objPath, verbose)
}

func readSymbolsWithNm(ctx context.Context, objPath string, verbose bool) ([]SymbolInfo, error) {
	out, err := utils.RunCommandOutput(ctx, "nm", "-g", objPath)
	if err != nil {
		return nil, fmt.Errorf("nm %s: %w", objPath, err)
	}
	return parseNmOutput(objPath, string(out)), nil
}

func readSymbolsWithObjdump(ctx context.Context, objPath string, verbose bool) ([]SymbolInfo, error) {
	out, err := utils.RunCommandOutput(ctx, "objdump", "-t", objPath)
	if err != nil {
		return nil, fmt.Errorf("objdump %s: %w", objPath, err)
	}
	return parseObjdumpOutput(objPath, string(out)), nil
}

func readSymbolsWithReadelf(ctx context.Context, objPath string, verbose bool) ([]SymbolInfo, error) {
	out, err := utils.RunCommandOutput(ctx, "readelf", "-s", objPath)
	if err != nil {
		return nil, fmt.Errorf("readelf %s: %w", objPath, err)
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
