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

package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/forgezero-cli/ForgeZero/cmd/fz/buildcmd"
	"github.com/forgezero-cli/ForgeZero/cmd/fz/cli"
	"github.com/forgezero-cli/ForgeZero/cmd/fz/stdio"
	"github.com/forgezero-cli/ForgeZero/cmd/fz/subcmd"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
	"github.com/forgezero-cli/ForgeZero/internal/builder"
	"github.com/forgezero-cli/ForgeZero/internal/compilecommands"
	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/contribute"
	"github.com/forgezero-cli/ForgeZero/internal/cplugin"
	"github.com/forgezero-cli/ForgeZero/internal/linker"
	"github.com/forgezero-cli/ForgeZero/internal/man"
	"github.com/forgezero-cli/ForgeZero/internal/reverse"
	"github.com/forgezero-cli/ForgeZero/internal/seal"
	"github.com/forgezero-cli/ForgeZero/internal/updater"
	"github.com/forgezero-cli/ForgeZero/internal/updater/rollback"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
	"github.com/forgezero-cli/ForgeZero/internal/watcher"
	"gopkg.in/yaml.v3"
)

func SetupTestMode() {
	utils.CheckToolFunc = func(string) error { return nil }
	assembler.SetRunCommand(func(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-o" {
				out := args[i+1]
				data := []byte("OBJ")
				_ = os.WriteFile(out, data, 0o755)
				return "", nil
			}
		}
		return "", nil
	})
	linker.SetRunner(buildcmd.FakeRunner{})
}

func HandleSeal() bool {
	for _, a := range os.Args[1:] {
		if a == "--seal" {
			if cli.IsTestMode() {
				stdio.WriteFmt(1, "%s\n", "seal written")
				return true
			}
			if err := seal.Seal(); err != nil {
				stdio.WriteFmt(2, "seal failed: %v\n", err)
				cli.Exit(2)
			}
			stdio.WriteFmt(1, "%s\n", "seal written")
			return true
		}
	}
	return false
}

func HandleSubcommands() bool {
	if len(os.Args) < 2 {
		return false
	}
	switch os.Args[1] {
	case "audit":
		subcmd.AuditMain(os.Args[2:])
		return true
	case "sbom":
		subcmd.SbomMain(os.Args[2:])
		return true
	case "verify":
		subcmd.VerifyMain(os.Args[2:])
		return true
	case "bench":
		subcmd.BenchMain(os.Args[2:])
		return true
	case "doctor":
		subcmd.DoctorMain(os.Args[2:])
		return true
	case "version":
		cli.OutputVersion()
		return true
	}
	return false
}

func HandleReverse(flags *cli.Flags) bool {
	if flags.OldReverseFile == "" {
		return false
	}
	cfg, err := reverse.ReverseFile(flags.OldReverseFile)
	if err != nil {
		stdio.WriteFmt(2, "reverse failed: %v\n", err)
		cli.Exit(2)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		stdio.WriteFmt(2, "failed to marshal config: %v\n", err)
		cli.Exit(2)
	}
	if err := os.WriteFile(".fz.yaml", data, 0o644); err != nil {
		stdio.WriteFmt(2, "failed to write .fz.yaml: %v\n", err)
		cli.Exit(2)
	}
	stdio.WriteFmt(1, "Generated .fz.yaml from %s\n", flags.OldReverseFile)
	return true
}

func HandleContribute(flags *cli.Flags) error {
	if flags.AsmPath != "" || flags.CcPath != "" || flags.DirPath != "" || flags.OutBin != "" || flags.Clean || flags.Watch || flags.UpdateMode || flags.ShellMode || flags.ShowMan || flags.ShowHelp {
		return stdio.Errorf("error: -contribute cannot be used with other flags\n")
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err := contribute.Run(context.Background(), root); err != nil {
		return err
	}
	stdio.WriteFmt(1, "CONTRIBUTING_USER.md generated successfully\n")
	return nil
}

func HandleUpdate(flags *cli.Flags) bool {
	if !flags.UpdateMode {
		return false
	}
	if err := updater.UpdateSelf(cli.VersionCore); err != nil {
		stdio.WriteFmt(2, "update failed: %v\n", err)
		os.Exit(1)
	}
	return true
}

func HandleRollback(flags *cli.Flags) bool {
	if flags.RollBackToFlag != "" {
		if err := rollback.To(flags.RollBackToFlag); err != nil {
			stdio.WriteFmt(2, "rollback failed: %v\n", err)
			os.Exit(1)
		}
		return true
	}
	if flags.RollBackFlag {
		if err := rollback.Run(); err != nil {
			stdio.WriteFmt(2, "rollback failed: %v\n", err)
			os.Exit(1)
		}
		return true
	}
	return false
}

func HandleVersion(flags *cli.Flags) bool {
	if !flags.ShowVersion && !flags.ShortVersion {
		return false
	}
	if flags.JSONOutput {
		report := stdio.BuildReport{Status: "info", ExitCode: 0, DurationMs: 0, Binary: cli.VersionCore}
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			stdio.WriteFmt(2, "version encode failed: %v\n", err)
			os.Exit(1)
		}
	}
	if flags.ShortVersion {
		stdio.WriteStdout(cli.VersionCore + "\n")
	} else {
		cli.OutputVersion()
	}
	os.Exit(0)
	return true
}

func HandleMan(flags *cli.Flags) bool {
	if !flags.ShowMan {
		return false
	}
	stdio.WriteFmt(1, "%s", man.GenerateManPage(cli.VersionCore))
	os.Exit(0)
	return true
}

func HandleHelp(flags *cli.Flags) bool {
	if !flags.ShowHelp {
		return false
	}
	cli.PrintHelp()
	os.Exit(0)
	return true
}

func HandleConfigError(err error, jsonOutput bool) {
	if jsonOutput {
		report := stdio.BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: err.Error()}
		if encErr := json.NewEncoder(os.Stdout).Encode(report); encErr != nil {
			stdio.WriteFmt(2, "config encode failed: %v\n", encErr)
			os.Exit(1)
		}
	} else {
		stdio.WriteFmt(2, "config error: %v\n", err)
	}
}

func GenerateCompileCommands(cfg *config.Config) error {
	dirs := []string{"."}
	if cfg != nil && len(cfg.SourceDirs) > 0 {
		dirs = cfg.SourceDirs
	} else if cfg != nil && cfg.SourceDir != "" {
		dirs = []string{cfg.SourceDir}
	}
	return compilecommands.Generate(cfg, dirs[0])
}

func HandleClean(flags *cli.Flags, cfg *config.Config) error {
	targetDir := flags.DirPath
	if targetDir == "" && cfg != nil && cfg.SourceDir != "" {
		targetDir = cfg.SourceDir
	}
	if targetDir == "" && cfg != nil && len(cfg.SourceDirs) > 0 {
		targetDir = cfg.SourceDirs[0]
	}
	if targetDir == "" {
		return stdio.Errorf("-clean requires -dir or source_dir/source_dirs in config")
	}
	return builder.CleanDir(targetDir, flags.Verbose)
}

func HandlePlugin(flags *cli.Flags, cfg *config.Config, srcPath string) error {
	goCtx := cplugin.GoContext{
		PluginPath: flags.PluginPath,
		ConfigPath: flags.ConfigPath,
		SourcePath: srcPath,
		DirPath:    flags.DirPath,
		OutBin:     flags.OutBin,
		OutObj:     flags.OutObj,
		BuildType:  flags.BuildType,
		Target:     flags.Target,
		Toolchain:  flags.Toolchain,
		Mode:       flags.Mode,
		CcFlags:    flags.CcFlags,
		LdFlags:    flags.LdFlags,
		Format:     flags.Format,
		Isolation:  flags.Isolation,
	}
	if cfg != nil {
		goCtx.SourceDirs = cfg.SourceDirs
	}
	m, err := cplugin.Load(flags.PluginPath)
	if err != nil {
		stdio.WriteStderr(err.Error() + "\n")
		return err
	}
	defer m.Close()
	if err := m.CallInitWithGoContext(goCtx); err != nil {
		stdio.WriteStderr(err.Error() + "\n")
		return err
	}
	return nil
}

func HandleWatch(flags *cli.Flags, cfg *config.Config, srcPath string, buildCtx buildcmd.BuildContext, timeoutSec int) {
	w, err := watcher.New()
	if err != nil {
		if flags.JSONOutput {
			report := stdio.BuildReport{Status: "error", ExitCode: 1, DurationMs: 0, Error: err.Error()}
			if encErr := json.NewEncoder(os.Stdout).Encode(report); encErr != nil {
				stdio.WriteFmt(2, "watch encode failed: %v\n", encErr)
			}
		} else {
			stdio.WriteFmt(2, "watcher error: %v\n", err)
		}
		os.Exit(1)
	}
	defer w.Close()

	watchTarget := flags.DirPath
	if srcPath != "" {
		watchTarget = filepath.Dir(srcPath)
	}
	if watchTarget == "" {
		if cfg != nil && len(cfg.SourceDirs) > 0 {
			watchTarget = cfg.SourceDirs[0]
		} else {
			watchTarget = "."
		}
	}

	if err := w.AddRecursive(watchTarget); err != nil {
		if flags.JSONOutput {
			report := stdio.BuildReport{Status: "error", ExitCode: 1, DurationMs: 0, Error: err.Error()}
			if encErr := json.NewEncoder(os.Stdout).Encode(report); encErr != nil {
				stdio.WriteFmt(2, "watch encode failed: %v\n", encErr)
			}
		} else {
			stdio.WriteFmt(2, "cannot watch: %v\n", err)
		}
		os.Exit(1)
	}

	if flags.ConfigPath != "" {
		if err := w.Add(flags.ConfigPath); err != nil {
			stdio.WriteFmt(2, "watch add failed: %v\n", err)
		}
	} else if cfgFile := config.DefaultConfigPath(); cfgFile != "" {
		if err := w.Add(cfgFile); err != nil {
			stdio.WriteFmt(2, "watch add failed: %v\n", err)
		}
	}

	if !flags.JSONOutput {
		stdio.WriteFmt(1, "Watching %s for changes...\n", watchTarget)
	}

	w.Watch(500*time.Millisecond, func(string) error {
		if !flags.JSONOutput {
			stdio.WriteFmt(1, "%s\n", "\nChange detected, rebuilding...")
		}
		ctx2, cancel2 := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
		defer cancel2()
		newResult := buildcmd.Build(ctx2, buildCtx, cfg)
		if newResult.Err != nil {
			if !flags.JSONOutput {
				stdio.WriteFmt(2, "rebuild failed: %v\n", newResult.Err)
			}
		} else if !flags.JSONOutput {
			stdio.WriteFmt(1, "%s\n", "Rebuild successful.")
		}
		return nil
	})

	select {}
}

func SetupAssemblerAndLinker(flags *cli.Flags) {
	assembler.ForceFASM = flags.ForceFASM
	if flags.RawFlag {
		flags.Mode = "raw"
	}
	if flags.ForceLdFlag {
		linker.ForceLD = true
	}

	assembler.CcFlags = flags.CcFlags
	linker.LdFlags = flags.LdFlags
	linker.Shared = flags.Shared
	assembler.Target = assembler.NormalizeTargetTriple(flags.Target)
	flags.Target = assembler.Target
	linker.SetTarget(flags.Target)

	if flags.LibMode {
		flags.BuildType = "static"
	}
	if flags.BuildType != "executable" && flags.BuildType != "static" {
		stdio.WriteFmt(2, "error: -type must be executable or static")
		cli.Exit(2)
	}

	if flags.Mode == "" {
		flags.Mode = "auto"
	}
	if flags.NoSanitize {
		flags.Sanitize = false
	}

	if flags.Jobs <= 0 {
		flags.Jobs = runtime.NumCPU()
	}

	if err := linker.SetOutputFormat(flags.Format); err != nil {
		stdio.WriteFmt(2, "error: %v\n", err)
		cli.Exit(2)
	}
	linker.LdScript = flags.LdScript
	linker.TextAddr = flags.TextAddr
	if flags.Linker != "" {
		linker.SetPreferredLinker(flags.Linker)
	}
}
