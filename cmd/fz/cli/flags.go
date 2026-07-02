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

package cli

import (
	"flag"
	"os"
)

func SetupFlags() *Flags {
	f := &Flags{}
	
	flag.BoolVar(&f.Watch, "watch", false, "")
	flag.BoolVar(&f.Sanitize, "sanitize", true, "")
	flag.BoolVar(&f.NoSanitize, "no-sanitize", false, "")
	flag.BoolVar(&f.Strict, "strict", false, "")
	flag.BoolVar(&f.JSONOutput, "json", false, "")
	flag.BoolVar(&f.ShowVersion, "v", false, "")
	flag.BoolVar(&f.ShowVersion, "version", false, "")
	flag.BoolVar(&f.ShortVersion, "v-short", false, "")
	flag.BoolVar(&f.ShowHelp, "h", false, "")
	flag.BoolVar(&f.ShowHelp, "help", false, "")
	flag.BoolVar(&f.ShowMan, "man", false, "")
	flag.StringVar(&f.Format, "format", "elf64", "")
	flag.BoolVar(&f.InitMode, "init", false, "initialize project: create .fz.yaml and .fzignore")
	flag.StringVar(&f.LdScript, "T", "", "linker script file (passed to ld via -T)")
	flag.StringVar(&f.TextAddr, "Ttext", "", "set text segment address (passed to ld)")
	flag.BoolVar(&f.ShellMode, "shell", false, "run interactive shell")
	flag.IntVar(&f.Jobs, "j", 1, "number of parallel jobs (0 = auto = CPU cores)")
	flag.BoolVar(&f.UpdateMode, "update", false, "update fz to the latest version")
	flag.BoolVar(&f.ContributeMode, "contribute", false, "generate CONTRIBUTING_USER.md and run contribute guidance")
	flag.StringVar(&f.BuildType, "type", "executable", "build type: executable (default) or static")
	flag.BoolVar(&f.LibMode, "lib", false, "build static library (archive)")
	flag.StringVar(&f.Target, "target", "x86_64-linux-gnu", "target triple")
	flag.StringVar(&f.Toolchain, "toolchain", "auto", "toolchain to use: auto or zig")
	flag.StringVar(&f.Isolation, "isolation", "none", "isolation level: none, standard, strict")
	flag.BoolVar(&f.GenCompileCommands, "compile-commands", false, "generate compile_commands.json for LSP and exit")
	flag.BoolVar(&f.Shared, "shared", false, "build shared library instead of executable")
	flag.StringVar(&f.CcFlags, "cc-flag", "", "additional C compiler flags (space-separated)")
	flag.StringVar(&f.LdFlags, "ld-flag", "", "additional linker flags (space-separated)")
	flag.BoolVar(&f.ForceFASM, "fasm", false, "use FASM instead of NASM for .asm files")
	flag.BoolVar(&f.RawFlag, "raw", false, "force raw linking (alias for -mode raw)")
	flag.BoolVar(&f.ForceLdFlag, "ld", false, "invoke ld directly, skip gcc/clang probes")
	flag.StringVar(&f.AsmPath, "asm", "", "path to .asm file")
	flag.StringVar(&f.CcPath, "cc", "", "path to C source file")
	flag.StringVar(&f.DirPath, "dir", "", "build all supported files recursively")
	flag.StringVar(&f.OutBin, "out", "", "output binary name")
	flag.StringVar(&f.OutObj, "out-obj", "", "object file name (single file only)")
	flag.StringVar(&f.Mode, "mode", "auto", "linking mode: auto, c, raw")
	flag.BoolVar(&f.Debug, "debug", false, "emit debug symbols (-g)")
	flag.BoolVar(&f.Debug, "d", false, "emit debug symbols (-g)")
	flag.BoolVar(&f.Verbose, "verbose", false, "print every command executed")
	flag.BoolVar(&f.KeepObj, "keep-obj", false, "keep temporary object files (when using -dir)")
	flag.BoolVar(&f.NoCache, "no-cache", false, "disable incremental cache")
	flag.BoolVar(&f.NoSymbolCheck, "no-symbol-check", false, "skip duplicate symbol pre-check")
	flag.StringVar(&f.ConfigPath, "config", "", "config file (default: .fz.yaml, fz.yaml, .fz.yml, fz.yml)")
	flag.StringVar(&f.PluginPath, "plugin", "", "shared object plugin file to load before build")
	flag.BoolVar(&f.Clean, "clean", false, "remove all build artifacts (.fz_objs, .fz_cache, binaries)")
	flag.StringVar(&f.GloriaPath, "gloria", "", "path to .glo file")
	flag.BoolVar(&f.AutoBuild, "autoBuild", false, "auto build project")
	flag.StringVar(&f.MuslOpt, "musl", "", "use static musl toolchain(e.g -musl=riscv64")
	flag.StringVar(&f.ProfileFlag, "profile", "balanced", "build profile for hardware")
	flag.StringVar(&f.ProfileFlag, "p", "balanced", "build profile (shorthand)")
	flag.BoolVar(&f.AlexMode, "alex", false, "run full test scanner projects for contribution")
	flag.BoolVar(&f.PyzeroFlag, "pyzero", false, "experimental: bump python format file to binaries")
	flag.StringVar(&f.OldReverseFile, "old-reverse", "", "generate .fz.yaml from legacy build files")
	flag.Var(&f.ISO, "iso", "package the build into a bootable ISO (optional dir: -iso=./isoroot)")
	flag.StringVar(&f.IsoOut, "iso-out", "", "output path for the generated ISO image")
	flag.BoolVar(&f.IsoHybrid, "iso-hybrid", false, "make the generated ISO hybrid (BIOS+USB bootable)")
	flag.BoolVar(&f.RollBackFlag, "rollback", false, "rollback version to old or stable(down to 1-2)")
	flag.StringVar(&f.RollBackToFlag, "rollback-to", "", "rollback-to needs version(e.g fz --rollback-to 5.1.0)")
	
	flag.Usage = func() {
		os.Stderr.WriteString("Run " + os.Args[0] + " -help for full usage.\n")
		os.Exit(2)
	}

	flag.Parse()
	
	return f
}

type Flags struct {
	Watch               bool
	Sanitize            bool
	NoSanitize          bool
	Strict              bool
	JSONOutput          bool
	ShowVersion         bool
	ShortVersion        bool
	ShowHelp            bool
	ShowMan             bool
	Format              string
	InitMode            bool
	LdScript            string
	TextAddr            string
	ShellMode           bool
	Jobs                int
	UpdateMode          bool
	ContributeMode      bool
	BuildType           string
	LibMode             bool
	Target              string
	Toolchain           string
	Isolation           string
	GenCompileCommands  bool
	Shared              bool
	CcFlags             string
	LdFlags             string
	ForceFASM           bool
	RawFlag             bool
	ForceLdFlag         bool
	AsmPath             string
	CcPath              string
	DirPath             string
	OutBin              string
	OutObj              string
	Mode                string
	Debug               bool
	Verbose             bool
	KeepObj             bool
	NoCache             bool
	NoSymbolCheck       bool
	ConfigPath          string
	PluginPath          string
	Clean               bool
	GloriaPath          string
	AutoBuild           bool
	MuslOpt             string
	ProfileFlag         string
	AlexMode            bool
	PyzeroFlag          bool
	OldReverseFile      string
	ISO                 ISOFlag
	IsoOut              string
	IsoHybrid           bool
	SourcePath          string

	RollBackFlag 		  bool 	 // fz --rollback 
	RollBackToFlag   string // fz --rollback-to <version>
}

