package linker

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"fz/internal/utils"
)

var (
	runner   CmdRunner = &RealCmdRunner{}
	LdScript string
	TextAddr string
)

func Link(ctx context.Context, obj, bin string, verbose bool, mode string, noSymbolCheck bool, sanitize bool, strict bool, libs []string) error {
	if err := utils.CheckFileExists(obj); err != nil {
		return err
	}
	info, err := os.Stat(obj)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fmt.Errorf("object file %s is empty", obj)
	}
	if err := utils.EnsureDir(bin); err != nil {
		return err
	}
	if !noSymbolCheck {
		if err := CheckDuplicateSymbols([]string{obj}, verbose); err != nil {
			return err
		}
	}

	switch runtime.GOOS {
	case "windows":
		return linkWindows(ctx, obj, bin, verbose, mode, noSymbolCheck, sanitize, strict, libs)
	default:
		switch mode {
		case "raw":
			if err := utils.CheckTool("ld"); err != nil {
				return err
			}
			return linkWithLd(ctx, obj, bin, verbose, libs)
		case "c":
			if err := utils.CheckTool("gcc"); err != nil {
				return err
			}
			return linkWithGcc(ctx, obj, bin, verbose, false, sanitize, strict, libs)
		case "auto":
			return tryAutoLink(ctx, obj, bin, verbose, sanitize, strict, libs)
		default:
			return fmt.Errorf("unsupported mode: %s (valid: auto, c, raw)", mode)
		}
	}
}

func LinkMultiple(ctx context.Context, objFiles []string, bin string, verbose bool, mode string, noSymbolCheck bool, sanitize bool, strict bool, libs []string) error {
	if len(objFiles) == 0 {
		return fmt.Errorf("no object files to link")
	}
	for _, obj := range objFiles {
		info, err := os.Stat(obj)
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			return fmt.Errorf("object file %s is empty", obj)
		}
	}
	if err := utils.EnsureDir(bin); err != nil {
		return err
	}
	if !noSymbolCheck {
		if err := CheckDuplicateSymbols(objFiles, verbose); err != nil {
			return err
		}
	}

	switch runtime.GOOS {
	case "windows":
		return linkMultipleWindows(ctx, objFiles, bin, verbose, mode, noSymbolCheck, sanitize, strict, libs)
	default:
		switch mode {
		case "raw":
			if err := utils.CheckTool("ld"); err != nil {
				return err
			}
			return linkMultipleWithLd(ctx, objFiles, bin, verbose, libs)
		case "c":
			if err := utils.CheckTool("gcc"); err != nil {
				return err
			}
			return linkMultipleWithGcc(ctx, objFiles, bin, verbose, false, sanitize, strict, libs)
		case "auto":
			return tryAutoLinkMultiple(ctx, objFiles, bin, verbose, sanitize, strict, libs)
		default:
			return fmt.Errorf("unsupported mode: %s (valid: auto, c, raw)", mode)
		}
	}
}
