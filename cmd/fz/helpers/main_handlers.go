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
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"time"

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
	linker.SetRunner(FakeRunner{})
}

func HandleSeal() bool {
	for _, a := range os.Args[1:] {
		if a == "--seal" {
			if IsTestMode() {
				WriteFmt(1, "%s\n", "seal written")
				return true
			}
			if err := seal.Seal(); err != nil {
				WriteFmt(2, "seal failed: %v\n", err)
				Exit(2)
			}
			WriteFmt(1, "%s\n", "seal written")
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
		AuditMain(os.Args[2:])
		return true
	case "sbom":
		SbomMain(os.Args[2:])
		return true
	case "verify":
		VerifyMain(os.Args[2:])
		return true
	case "bench":
		BenchMain(os.Args[2:])
		return true
	case "doctor":
		DoctorMain(os.Args[2:])
		return true
	case "version":
		OutputVersion()
		return true
	}
	return false
}

func HandleReverse(flags *Flags) bool {
	if flags.OldReverseFile == "" {
		return false
	}
	cfg, err := reverse.ReverseFile(flags.OldReverseFile)
	if err != nil {
		WriteFmt(2, "reverse failed: %v\n", err)
		Exit(2)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		WriteFmt(2, "failed to marshal config: %v\n", err)
		Exit(2)
	}
	if err := os.WriteFile(".fz.yaml", data, 0o644); err != nil {
		WriteFmt(2, "failed to write .fz.yaml: %v\n", err)
		Exit(2)
	}
	WriteFmt(1, "Generated .fz.yaml from %s\n", flags.OldReverseFile)
	return true
}

func HandleContribute(flags *Flags) error {
	if flags.AsmPath != "" || flags.CcPath != "" || flags.DirPath != "" || flags.OutBin != "" || flags.Clean || flags.Watch || flags.UpdateMode || flags.ShellMode || flags.ShowMan || flags.ShowHelp {
		return Errorf("error: -contribute cannot be used with other flags\n")
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err := contribute.Run(context.Background(), root); err != nil {
		return err
	}
	WriteFmt(1, "CONTRIBUTING_USER.md generated successfully\n")
	return nil
}

func HandleUpdate(flags *Flags) bool {
	if !flags.UpdateMode {
		return false
	}
	if err := updater.UpdateSelf(VersionCore); err != nil {
		WriteFmt(2, "update failed: %v\n", err)
		os.Exit(1)
	}
	return true
}

func HandleVersion(flags *Flags) bool {
	if !flags.ShowVersion {
		return false
	}
	if flags.JSONOutput {
		report := BuildReport{Status: "info", ExitCode: 0, DurationMs: 0, Binary: VersionCore}
		_ = json.NewEncoder(os.Stdout).Encode(report)
	}
	isShort := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "v" {
			isShort = true
		}
	})
	if isShort {
		WriteStdout(VersionCore + "\n")
	} else {
		OutputVersion()
	}
	os.Exit(0)
	return true
}

func HandleMan(flags *Flags) bool {
	if !flags.ShowMan {
		return false
	}
	WriteFmt(1, "%s", man.GenerateManPage(VersionCore))
	os.Exit(0)
	return true
}

func HandleHelp(flags *Flags) bool {
	if !flags.ShowHelp {
		return false
	}
	PrintHelp()
	os.Exit(0)
	return true
}

func HandleSourceError(err error, jsonOutput bool) {
	if jsonOutput {
		report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: err.Error()}
		_ = json.NewEncoder(os.Stdout).Encode(report)
	} else {
		WriteFmt(2, "%s\n", err.Error())
	}
}

func HandleConfigError(err error, jsonOutput bool) {
	if jsonOutput {
		report := BuildReport{Status: "error", ExitCode: 2, DurationMs: 0, Error: err.Error()}
		_ = json.NewEncoder(os.Stdout).Encode(report)
	} else {
		WriteFmt(2, "config error: %v\n", err)
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

func HandleClean(flags *Flags, cfg *config.Config) error {
	targetDir := flags.DirPath
	if targetDir == "" && cfg != nil && cfg.SourceDir != "" {
		targetDir = cfg.SourceDir
	}
	if targetDir == "" && cfg != nil && len(cfg.SourceDirs) > 0 {
		targetDir = cfg.SourceDirs[0]
	}
	if targetDir == "" {
		return Errorf("-clean requires -dir or source_dir/source_dirs in config")
	}
	return builder.CleanDir(targetDir, flags.Verbose)
}

func HandlePlugin(flags *Flags, cfg *config.Config, srcPath string) error {
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
		WriteStderr(err.Error() + "\n")
		return err
	}
	defer m.Close()
	if err := m.CallInitWithGoContext(goCtx); err != nil {
		WriteStderr(err.Error() + "\n")
		return err
	}
	_, err = cplugin.Load(flags.PluginPath)
	if err != nil {
		WriteStderr(err.Error() + "\n")
		return err
	}
	return nil
}

func HandleWatch(flags *Flags, cfg *config.Config, srcPath string, buildCtx BuildContext, timeoutSec int) {
	w, err := watcher.New()
	if err != nil {
		if flags.JSONOutput {
			report := BuildReport{Status: "error", ExitCode: 1, DurationMs: 0, Error: err.Error()}
			_ = json.NewEncoder(os.Stdout).Encode(report)
		} else {
			WriteFmt(2, "watcher error: %v\n", err)
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
			report := BuildReport{Status: "error", ExitCode: 1, DurationMs: 0, Error: err.Error()}
			_ = json.NewEncoder(os.Stdout).Encode(report)
		} else {
			WriteFmt(2, "cannot watch: %v\n", err)
		}
		os.Exit(1)
	}

	if flags.ConfigPath != "" {
		_ = w.Add(flags.ConfigPath)
	} else if cfgFile := config.DefaultConfigPath(); cfgFile != "" {
		_ = w.Add(cfgFile)
	}

	if !flags.JSONOutput {
		WriteFmt(1, "Watching %s for changes...\n", watchTarget)
	}

	w.Watch(500*time.Millisecond, func(string) error {
		if !flags.JSONOutput {
			WriteFmt(1, "%s\n", "\nChange detected, rebuilding...")
		}
		ctx2, cancel2 := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
		defer cancel2()
		newResult := Build(ctx2, buildCtx, cfg)
		if newResult.Err != nil {
			if !flags.JSONOutput {
				WriteFmt(2, "rebuild failed: %v\n", newResult.Err)
			}
		} else if !flags.JSONOutput {
			WriteFmt(1, "%s\n", "Rebuild successful.")
		}
		return nil
	})

	select {}
}

func SetupAssemblerAndLinker(flags *Flags) {
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
		WriteFmt(2, "error: -type must be executable or static")
		Exit(2)
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
		WriteFmt(2, "error: %v\n", err)
		Exit(2)
	}
	linker.LdScript = flags.LdScript
	linker.TextAddr = flags.TextAddr
}

func ValidateSourceFlags(flags *Flags) (string, error) {
	srcProvided := 0
	var srcPath string

	if flags.AsmPath != "" {
		srcProvided++
		srcPath = flags.AsmPath
	}
	if flags.CcPath != "" {
		srcProvided++
		srcPath = flags.CcPath
	}
	if flags.GloriaPath != "" {
		srcProvided++
		srcPath = flags.GloriaPath
	}
	if flags.DirPath != "" {
		srcProvided++
	}

	if srcProvided == 0 {
		return "", Errorf("missing source: use -asm, -cc, -dir, or config")
	}

	if srcProvided > 1 {
		return "", Errorf("specify only one of -asm, -cc, -gloria or -dir")
	}

	return srcPath, nil
}