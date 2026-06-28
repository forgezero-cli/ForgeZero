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

package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/forgezero-cli/ForgeZero/cmd/fz/helpers"
	"github.com/forgezero-cli/ForgeZero/internal/assembler"
	initpkg "github.com/forgezero-cli/ForgeZero/internal/init"
	"github.com/forgezero-cli/ForgeZero/internal/linker"
	"github.com/forgezero-cli/ForgeZero/internal/scripts"
	"github.com/forgezero-cli/ForgeZero/internal/shell"
	"github.com/forgezero-cli/ForgeZero/internal/testrunner"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(helpers.ExitPanic); ok {
				return
			}
			panic(r)
		}
	}()

	if helpers.IsTestMode() {
		helpers.SetupTestMode()
	}

	if helpers.HandleSeal() {
		return
	}

	if err := utils.SelfAttest(); err != nil {
		helpers.WriteFmt(2, "self-attestation failed: %v\n", err)
		os.Exit(1)
	}

	if helpers.HandleSubcommands() {
		return
	}

	flags := helpers.SetupFlags()

	if helpers.HandleReverse(flags) {
		return
	}

	helpers.SetupProfile(flags)

	if flags.AlexMode {
		if err := testrunner.RunSuite(flags.Verbose, flags.JSONOutput, flags.AlexMode); err != nil {
			helpers.WriteFmt(2, "test suite failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	muslCtx := helpers.SetupMusl(flags.MuslOpt)

	if !helpers.ValidateAll(flags) {
		helpers.Exit(2)
	}

	if flags.InitMode {
		if err := initpkg.Run(); err != nil {
			helpers.WriteFmt(2, "init failed: %v\n", err)
			os.Exit(1)
		}
		helpers.WriteFmt(1, "%s\n", "project initialized. edit .fz.yaml to configure ur build.")
		return
	}

	if flags.ContributeMode {
		if err := helpers.HandleContribute(flags); err != nil {
			helpers.WriteFmt(2, "contribute failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	ctx := context.Background()
	if muslCtx.Use {
		ctx = context.WithValue(ctx, utils.TargetCtxKey, assembler.Target)
	}

	if len(os.Args) >= 2 && os.Args[1] == "pm" {
		helpers.HandlePackageManager(ctx, os.Args)
		return
	}

	if flags.AutoBuild {
		if err := linker.AutoBuildProject(ctx); err != nil {
			helpers.WriteStderr("auto build failed!\n")
		}
	}

	helpers.SetupAssemblerAndLinker(flags)

	if helpers.HandleUpdate(flags) {
		return
	}

	if flags.ShellMode {
		shell.Run()
		return
	}

	if helpers.HandleMan(flags) {
		return
	}

	if helpers.HandleHelp(flags) {
		return
	}

	if helpers.HandleVersion(flags) {
		return
	}

	root, err := os.Getwd()
	if err == nil {
		utils.SetExecutionRoot(root)
	}

	if flags.Watch && flags.JSONOutput {
		helpers.WriteFmt(2, "%s\n", "error: -watch and -json cannot be used together")
		helpers.Exit(2)
	}

	cfg, srcPath, err := helpers.LoadConfig(flags)
	if err != nil {
		helpers.HandleConfigError(err, flags.JSONOutput)
		helpers.Exit(2)
	}

	if flags.GloriaPath != "" {
		if err := helpers.ProcessGloria(flags.GloriaPath, flags.OutBin); err != nil {
			helpers.WriteFmt(2, "%v\n", err)
			helpers.Exit(2)
		}
		os.Exit(0)
	}

	helpers.ApplyConfigToFlags(cfg, flags)
	helpers.SetupZig(cfg, flags.Toolchain)

	if flags.GenCompileCommands {
		if err := helpers.GenerateCompileCommands(cfg); err != nil {
			helpers.WriteFmt(2, "error generating compile_commands.json: %v\n", err)
			os.Exit(1)
		}
		helpers.WriteFmt(1, "%s\n", "compile_commands.json generated")
		return
	}

	if flags.Clean {
		if err := helpers.HandleClean(flags, cfg); err != nil {
			if flags.JSONOutput {
				report := helpers.BuildReport{Status: "error", ExitCode: 1, DurationMs: 0, Error: err.Error()}
				_ = json.NewEncoder(os.Stdout).Encode(report)
			}
			os.Exit(1)
		}
		if flags.JSONOutput {
			report := helpers.BuildReport{Status: "success", ExitCode: 0, DurationMs: 0, Binary: "cleaned"}
			_ = json.NewEncoder(os.Stdout).Encode(report)
		}
		return
	}

	if flags.PluginPath != "" {
		if err := helpers.HandlePlugin(flags, cfg, srcPath); err != nil {
			os.Exit(1)
		}
	}

	assembler.OutputFormat = flags.Format

	timeoutSec := 120
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	if cfg != nil {
		ctx = utils.ContextWithConfig(ctx, cfg)
	}

	buildCtx := helpers.BuildContext{
		SrcPath:       srcPath,
		DirPath:       flags.DirPath,
		OutBin:        flags.OutBin,
		OutObj:        flags.OutObj,
		Mode:          flags.Mode,
		Debug:         flags.Debug,
		Verbose:       flags.Verbose,
		KeepObj:       flags.KeepObj,
		NoCache:       flags.NoCache,
		NoSymbolCheck: flags.NoSymbolCheck,
		Sanitize:      flags.Sanitize,
		Strict:        flags.Strict,
		Format:        flags.Format,
		Jobs:          flags.Jobs,
		BuildType:     flags.BuildType,
		JSONOutput:    flags.JSONOutput,
		MuslCtx:       muslCtx,
	}

	result := helpers.Build(ctx, buildCtx, cfg)

	if result.Err == nil && cfg != nil && len(cfg.Scripts) > 0 {
		scriptsConfig := &scripts.ScriptsConfigure{
			Commands: cfg.Scripts,
			Verbose:  flags.Verbose,
		}
		if err := scriptsConfig.Run(ctx); err != nil {
			helpers.WriteFmt(2, "script failed: %v\n", err)
			os.Exit(1)
		}
	}

	if result.Err != nil {
		if flags.JSONOutput {
			report := helpers.BuildReport{
				Status:      "error",
				ExitCode:    1,
				DurationMs:  result.DurationMs,
				Binary:      result.Binary,
				SourceFiles: result.SourceFiles,
				ObjectFiles: result.ObjectFiles,
				Error:       result.Err.Error(),
			}
			_ = json.NewEncoder(os.Stdout).Encode(report)
		} else {
			helpers.WriteFmt(2, "build failed: %v\n", result.Err)
		}
		if !flags.Watch {
			os.Exit(1)
		}
	} else if flags.JSONOutput {
		report := helpers.BuildReport{
			Status:      "success",
			ExitCode:    0,
			DurationMs:  result.DurationMs,
			Binary:      result.Binary,
			SourceFiles: result.SourceFiles,
			ObjectFiles: result.ObjectFiles,
		}
		_ = json.NewEncoder(os.Stdout).Encode(report)
	} else if !flags.JSONOutput && result.Binary != "" {
		helpers.WriteFmt(1, "Built: %s\n", result.Binary)
	}

	if flags.Watch {
		helpers.HandleWatch(flags, cfg, srcPath, buildCtx, timeoutSec)
	}
}