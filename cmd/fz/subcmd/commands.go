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

package subcmd

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/forgezero-cli/ForgeZero/cmd/fz/cli"
	"github.com/forgezero-cli/ForgeZero/cmd/fz/stdio"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
	"github.com/forgezero-cli/ForgeZero/internal/audit"
	"github.com/forgezero-cli/ForgeZero/internal/bench"
	"github.com/forgezero-cli/ForgeZero/internal/builder"
	"github.com/forgezero-cli/ForgeZero/internal/config"
	"github.com/forgezero-cli/ForgeZero/internal/doctor"
	"github.com/forgezero-cli/ForgeZero/internal/linker"
	"github.com/forgezero-cli/ForgeZero/internal/sbom"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
	"github.com/forgezero-cli/ForgeZero/internal/verify"
)

func AuditMain(args []string) {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	configPath := fs.String("config", "", "config file path")
	jsonOutput := fs.Bool("json", false, "machine-readable output")
	verbose := fs.Bool("verbose", false, "print verbose audit output")
	vendorDir := fs.String("vendor", "vendor", "vendor directory to scan")
	if err := fs.Parse(args); err != nil {
		stdio.WriteFmt(2, "audit failed: %v\n", err)
		os.Exit(1)
	}

	root, err := os.Getwd()
	if err != nil {
		stdio.WriteFmt(2, "audit failed: %v\n", err)
		os.Exit(1)
	}
	utils.SetExecutionRoot(root)
	var cfg *config.Config
	if *configPath != "" {
		cfg, err = config.Load(*configPath)
	} else {
		cfg, err = config.LoadMerged("")
	}
	if err != nil {
		stdio.WriteFmt(2, "audit failed: %v\n", err)
		os.Exit(1)
	}
	if cfg != nil {
		for k, v := range cfg.ToolChecksums {
			utils.ToolChecksums.Store(k, v)
		}
	}
	if *verbose {
		stdio.WriteFmt(2, "audit: scanning project root %s using vendor dir %s\n", root, *vendorDir)
	}
	result, err := audit.ScanProject(context.Background(), root, *vendorDir, cfg)
	if err != nil {
		stdio.WriteFmt(2, "audit failed: %v\n", err)
		os.Exit(1)
	}
	if len(result.Findings) == 0 {
		if *jsonOutput {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "clean", "findings": []any{}})
		} else {
			stdio.WriteFmt(1, "%s\n", "audit passed: no vulnerabilities found")
		}
		return
	}
	if *jsonOutput {
		_ = json.NewEncoder(os.Stdout).Encode(result)
		return
	}
	for _, finding := range result.Findings {
		stdio.WriteFmt(1, "[%s] %s\n", finding.Package, finding.Summary)
		stdio.WriteFmt(1, "  path: %s\n", finding.Path)
		if finding.Version != "" {
			stdio.WriteFmt(1, "  version: %s\n", finding.Version)
		}
		if finding.URL != "" {
			stdio.WriteFmt(1, "  url: %s\n", finding.URL)
		}
	}
	os.Exit(1)
}

func SbomMain(args []string) {
	fs := flag.NewFlagSet("sbom", flag.ExitOnError)
	configPath := fs.String("config", "", "config file path")
	jsonOutput := fs.Bool("json", false, "machine-readable output")
	verbose := fs.Bool("verbose", false, "print verbose sbom generation output")
	vendorDir := fs.String("vendor", "vendor", "vendor directory to scan")
	outPath := fs.String("out", "sbom.json", "output SBOM file path")
	target := fs.String("target", "", "target triple to annotate in SBOM")
	if err := fs.Parse(args); err != nil {
		stdio.WriteFmt(2, "sbom failed: %v\n", err)
		os.Exit(2)
	}
	root, err := os.Getwd()
	if err != nil {
		stdio.WriteFmt(2, "sbom failed: %v\n", err)
		os.Exit(1)
	}
	utils.SetExecutionRoot(root)
	var cfg *config.Config
	if *configPath != "" {
		cfg, err = config.Load(*configPath)
	} else {
		cfg, err = config.LoadMerged("")
	}
	if err != nil {
		stdio.WriteFmt(2, "sbom failed: %v\n", err)
		os.Exit(1)
	}
	if cfg != nil {
		for k, v := range cfg.ToolChecksums {
			utils.ToolChecksums.Store(k, v)
		}
	}
	if *verbose {
		stdio.WriteFmt(2, "sbom: generating SBOM for project root %s using vendor dir %s\n", root, *vendorDir)
	}
	doc, err := sbom.Generate(root, *vendorDir, cli.VersionCore, cfg, *target)
	if err != nil {
		stdio.WriteFmt(2, "sbom failed: %v\n", err)
		os.Exit(1)
	}
	data, err := sbom.Marshal(doc)
	if err != nil {
		stdio.WriteFmt(2, "sbom failed: %v\n", err)
		os.Exit(1)
	}
	if *jsonOutput {
		stdio.WriteFmt(1, "%s\n", string(data))
		return
	}
	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		stdio.WriteFmt(2, "sbom failed: %v\n", err)
		os.Exit(1)
	}
	if *verbose {
		stdio.WriteFmt(2, "sbom written to %s\n", *outPath)
	}
}

func DoctorMain(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "machine-readable output")
	rootPath := fs.String("root", "", "project root (default: cwd)")
	if err := fs.Parse(args); err != nil {
		stdio.WriteFmt(2, "doctor failed: %v\n", err)
		os.Exit(2)
	}
	root := *rootPath
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			stdio.WriteFmt(2, "doctor failed: %v\n", err)
			os.Exit(1)
		}
		root = cwd
	}
	report, err := doctor.Run(context.Background(), doctor.Options{Root: root})
	if err != nil {
		stdio.WriteFmt(2, "doctor failed: %v\n", err)
		os.Exit(1)
	}
	if *jsonOutput {
		data, merr := doctor.MarshalJSON(report)
		if merr != nil {
			stdio.WriteFmt(2, "doctor failed: %v\n", merr)
			os.Exit(1)
		}
		stdio.WriteFmt(1, "%s\n", string(data))
	} else {
		stdio.WriteFmt(1, "%s", doctor.FormatHuman(report))
	}
	if !report.Healthy {
		os.Exit(1)
	}
}

func VerifyMain(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	rootPath := fs.String("root", ".", "project root directory")
	manifestPath := fs.String("manifest", "blake3.manifest", "manifest file path")
	updateManifest := fs.Bool("update", false, "update manifest file")
	jsonOutput := fs.Bool("json", false, "machine-readable output")
	if err := fs.Parse(args); err != nil {
		stdio.WriteFmt(2, "verify failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(*rootPath); err != nil {
		stdio.WriteFmt(2, "verify failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(*manifestPath); err != nil {
		stdio.WriteFmt(2, "verify failed: %v\n", err)
		os.Exit(2)
	}
	root := filepath.Clean(*rootPath)
	manifest := filepath.Clean(*manifestPath)
	if *updateManifest {
		if err := verify.WriteManifest(manifest, root); err != nil {
			stdio.WriteFmt(2, "verify failed: %v\n", err)
			os.Exit(1)
		}
		if *jsonOutput {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "updated", "manifest": manifest})
			return
		}
		stdio.WriteFmt(1, "manifest updated: %s\n", manifest)
		return
	}
	result, err := verify.VerifyRoot(root, manifest)
	if err != nil {
		stdio.WriteFmt(2, "verify failed: %v\n", err)
		os.Exit(1)
	}
	if len(result.Missing) == 0 && len(result.Modified) == 0 && len(result.Extra) == 0 {
		if *jsonOutput {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "clean"})
			return
		}
		stdio.WriteFmt(1, "%s\n", "verify passed: source tree integrity intact")
		return
	}
	if *jsonOutput {
		_ = json.NewEncoder(os.Stdout).Encode(result)
		os.Exit(1)
	}
	if len(result.Missing) > 0 {
		stdio.WriteFmt(1, "%s\n", "missing files:")
		for _, path := range result.Missing {
			stdio.WriteFmt(1, "  %s\n", path)
		}
	}
	if len(result.Modified) > 0 {
		stdio.WriteFmt(1, "%s\n", "modified files:")
		for _, path := range result.Modified {
			stdio.WriteFmt(1, "  %s\n", path)
		}
	}
	if len(result.Extra) > 0 {
		stdio.WriteFmt(1, "%s\n", "extra files:")
		for _, path := range result.Extra {
			stdio.WriteFmt(1, "  %s\n", path)
		}
	}
	os.Exit(1)
}

func BenchMain(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	asmPath := fs.String("asm", "", "assembler source file")
	ccPath := fs.String("cc", "", "C source file")
	dirPath := fs.String("dir", "", "source directory")
	outBin := fs.String("out", "bench-out", "output binary path")
	mode := fs.String("mode", "auto", "link mode: auto, c, raw")
	target := fs.String("target", "x86_64-linux-gnu", "target triple")
	toolchain := fs.String("toolchain", "auto", "toolchain: auto or zig")
	jsonOutput := fs.Bool("json", false, "machine-readable output")
	verbose := fs.Bool("verbose", false, "verbose output")
	timeoutSec := fs.Int("timeout", 60, "timeout in seconds")
	jobs := fs.Int("j", 0, "parallel jobs")
	if err := fs.Parse(args); err != nil {
		stdio.WriteFmt(2, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(*asmPath); err != nil {
		stdio.WriteFmt(2, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(*ccPath); err != nil {
		stdio.WriteFmt(2, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(*dirPath); err != nil {
		stdio.WriteFmt(2, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIPath(*outBin); err != nil {
		stdio.WriteFmt(2, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIArg(*mode); err != nil {
		stdio.WriteFmt(2, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIArg(*target); err != nil {
		stdio.WriteFmt(2, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if err := utils.ValidateCLIArg(*toolchain); err != nil {
		stdio.WriteFmt(2, "bench failed: %v\n", err)
		os.Exit(2)
	}
	if *mode != "auto" && *mode != "c" && *mode != "raw" {
		stdio.WriteFmt(2, "%s\n", "bench failed: invalid mode")
		os.Exit(2)
	}
	if *toolchain != "auto" && *toolchain != "zig" {
		stdio.WriteFmt(2, "%s\n", "bench failed: invalid toolchain")
		os.Exit(2)
	}
	if *jobs <= 0 {
		*jobs = runtime.NumCPU()
	}
	if *asmPath == "" && *ccPath == "" && *dirPath == "" {
		stdio.WriteFmt(2, "%s\n", "bench failed: missing source path")
		os.Exit(2)
	}
	if *asmPath != "" && *ccPath != "" {
		stdio.WriteFmt(2, "%s\n", "bench failed: specify only one of -asm or -cc")
		os.Exit(2)
	}
	root, err := os.Getwd()
	if err != nil {
		stdio.WriteFmt(2, "bench failed: %v\n", err)
		os.Exit(1)
	}
	utils.SetExecutionRoot(root)
	if *toolchain == "zig" {
		assembler.ZigRequested = true
		linker.ZigRequested = true
	}

	if utils.CheckTool("zig") == nil {
		assembler.ZigEnabled = true
		linker.ZigEnabled = true
	}
	if *dirPath != "" {
		*dirPath = filepath.Clean(*dirPath)
	}
	if *asmPath != "" {
		*asmPath = filepath.Clean(*asmPath)
	}
	if *ccPath != "" {
		*ccPath = filepath.Clean(*ccPath)
	}
	if *outBin != "" {
		*outBin = filepath.Clean(*outBin)
	}
	benchTimer := bench.NewTimer()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSec)*time.Second)
	defer cancel()
	if *asmPath != "" {
		objName := strings.TrimSuffix(filepath.Base(*asmPath), filepath.Ext(*asmPath)) + ".o"
		if err := benchTimer.Stage("assemble", func() error {
			return assembler.Assemble(ctx, *asmPath, objName, false, *verbose, *mode)
		}); err != nil {
			stdio.WriteFmt(2, "bench failed: %v\n", err)
			os.Exit(1)
		}
		if err := benchTimer.Stage("link", func() error {
			return linker.Link(ctx, objName, *outBin, *verbose, *mode, false, true, false, nil)
		}); err != nil {
			stdio.WriteFmt(2, "bench failed: %v\n", err)
			os.Exit(1)
		}
	} else if *ccPath != "" {
		objName := strings.TrimSuffix(filepath.Base(*ccPath), filepath.Ext(*ccPath)) + ".o"
		if err := benchTimer.Stage("compile", func() error {
			return assembler.Assemble(ctx, *ccPath, objName, false, *verbose, *mode)
		}); err != nil {
			stdio.WriteFmt(2, "bench failed: %v\n", err)
			os.Exit(1)
		}
		if err := benchTimer.Stage("link", func() error {
			return linker.Link(ctx, objName, *outBin, *verbose, *mode, false, true, false, nil)
		}); err != nil {
			stdio.WriteFmt(2, "bench failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := benchTimer.Stage("build_directory", func() error {
			_, err := builder.BuildDir(ctx, []string{*dirPath}, *outBin, false, *verbose, *mode, false, false, false, true, false, nil, nil, nil, nil, nil, *jobs, "executable")
			return err
		}); err != nil {
			stdio.WriteFmt(2, "bench failed: %v\n", err)
			os.Exit(1)
		}
	}
	err = benchTimer.Error()
	if err != nil {
		stdio.WriteFmt(2, "bench failed: %v\n", err)
		os.Exit(1)
	}
	if *jsonOutput {
		data, _ := benchTimer.JSON()
		stdio.WriteFmt(1, "%s\n", string(data))
		return
	}
	stdio.WriteFmt(1, "%s", benchTimer.Report())
}