/*
 * Copyright (c) 2026 ForgeZero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package helpers

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
	"github.com/forgezero-cli/ForgeZero/internal/builder"
	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/ignore"
	"github.com/forgezero-cli/ForgeZero/internal/linker"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

type BuildResult struct {
	Binary      string
	ObjectFiles []string
	SourceFiles []string
	DurationMs  int64
	Err         error
}

func Build(ctx context.Context, buildCtx BuildContext, cfg *config.Config) BuildResult {
	startTime := time.Now()
	result := BuildResult{}

	if buildCtx.SrcPath != "" {
		result.SourceFiles = append(result.SourceFiles, buildCtx.SrcPath)

		if err := utils.CheckFileExists(buildCtx.SrcPath); err != nil {
			result.Err = err
			return result
		}

		ext := filepath.Ext(buildCtx.SrcPath)
		if !utils.SupportedExtension(ext) {
			result.Err = Errorf("unsupported extension: %s", ext)
			return result
		}

		binName, objName := utils.DeriveNames(buildCtx.SrcPath, buildCtx.OutBin, buildCtx.OutObj)
		if buildCtx.Format == "bin" {
			objName = binName
		}

		result.ObjectFiles = append(result.ObjectFiles, objName)
		result.Binary = binName

		if buildCtx.Verbose && !buildCtx.JSONOutput {
			if ext == ".c" || ext == ".cpp" || ext == ".m" || ext == ".cc" || ext == ".cxx" {
				WriteFmt(1, "Compiling %s -> %s\n", buildCtx.SrcPath, objName)
			} else {
				WriteFmt(1, "Assembling %s -> %s\n", buildCtx.SrcPath, objName)
			}
		}

		if err := assembler.Assemble(ctx, buildCtx.SrcPath, objName, buildCtx.Debug, buildCtx.Verbose, buildCtx.Mode); err != nil {
			result.Err = err
			return result
		}

		if buildCtx.MuslCtx.Use {
			if err := BuildWithMusl([]string{objName}, binName, buildCtx.MuslCtx.Arch, buildCtx.Verbose, buildCtx.JSONOutput); err != nil {
				result.Err = err
				return result
			}
			result.DurationMs = time.Since(startTime).Milliseconds()
			return result
		}

		if buildCtx.Format == "bin" {
			if objName != binName {
				if err := linker.Link(ctx, objName, binName, buildCtx.Verbose, buildCtx.Mode, buildCtx.NoSymbolCheck, buildCtx.Sanitize, buildCtx.Strict, nil); err != nil {
					result.Err = err
					return result
				}
			}
			if !buildCtx.JSONOutput {
				if !buildCtx.Verbose {
					assembler.WriteFlatAssembledNotice(binName)
				}
				WriteFmt(1, "Built: %s\n", binName)
			}
			result.DurationMs = time.Since(startTime).Milliseconds()
			return result
		}

		if buildCtx.Verbose && !buildCtx.JSONOutput {
			WriteFmt(1, "Linking %s -> %s (mode: %s)\n", objName, binName, buildCtx.Mode)
		}

		if err := linker.Link(ctx, objName, binName, buildCtx.Verbose, buildCtx.Mode, buildCtx.NoSymbolCheck, buildCtx.Sanitize, buildCtx.Strict, nil); err != nil {
			result.Err = err
			return result
		}

		if !buildCtx.JSONOutput {

		}

		result.DurationMs = time.Since(startTime).Milliseconds()
		return result
	}

	if buildCtx.DirPath != "" {
		if buildCtx.Format == "bin" {
			result.Err = Errorf("-format bin is not supported for directory builds")
			return result
		}

		dirs := []string{buildCtx.DirPath}

		for _, d := range dirs {
			info, err := os.Stat(d)
			if err != nil {
				result.Err = err
				return result
			}
			if !info.IsDir() {
				result.Err = Errorf("%s is not a directory", d)
				return result
			}
		}

		if buildCtx.OutBin != "" {
			if st, err := os.Stat(buildCtx.OutBin); err == nil && st.IsDir() {
				result.Err = Errorf("output path %s is a directory", buildCtx.OutBin)
				return result
			}
		}

		var exclude, includes, sourceFilesList, libs []string
		var ignoreMatcher *ignore.IgnoreMatcher

		if cfg != nil {
			exclude = cfg.Exclude
			includes = cfg.Include
			sourceFilesList = cfg.SourceFiles
			libs = cfg.Libs

			if cfg.IgnoreFile != "" {
				if _, err := os.Stat(cfg.IgnoreFile); err == nil {
					if ignoreMatcher, err = ignore.LoadIgnoreFile(cfg.IgnoreFile); err != nil && buildCtx.Verbose {
						WriteFmt(1, "warning: cannot load ignore file %s: %v\n", cfg.IgnoreFile, err)
					}
				}
			}
		}

		muslCtx := buildCtx.MuslCtx
		if muslCtx.Use {
			buildCtx.KeepObj = true
			buildCtx.NoCache = true
			buildCtx.BuildType = "obj"
		}

		res, err := builder.BuildDir(ctx, dirs, buildCtx.OutBin, buildCtx.Debug, buildCtx.Verbose,
			buildCtx.Mode, buildCtx.KeepObj, buildCtx.NoCache, buildCtx.NoSymbolCheck,
			buildCtx.Sanitize, buildCtx.Strict, exclude, sourceFilesList, ignoreMatcher,
			includes, libs, buildCtx.Jobs, buildCtx.BuildType)

		if err != nil {
			result.Err = err
			return result
		}

		result.ObjectFiles = res.ObjectFiles
		result.Binary = res.Binary

		if muslCtx.Use {
			if err := BuildWithMusl(res.ObjectFiles, res.Binary, muslCtx.Arch, buildCtx.Verbose, buildCtx.JSONOutput); err != nil {
				result.Err = err
				return result
			}
		}

		if !buildCtx.JSONOutput {
			if !buildCtx.KeepObj && buildCtx.Verbose {
				WriteFmt(1, "Removed object dir: %s\n", res.ObjDir)
			}
			WriteFmt(1, "Built: %s\n", res.Binary)
		}

		result.DurationMs = time.Since(startTime).Milliseconds()
		return result
	}

	result.Err = Errorf("no source to build")
	return result
}