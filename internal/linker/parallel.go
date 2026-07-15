/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version of the License.
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
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/drivers/scheduler"
	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

const (
	minParallelLinkObjects = 24
	minPartitionSize       = 16
)

type LinkTarget struct {
	Name string
	Objs []string
}

func isArchiveFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".a", ".lib", ".so", ".dylib":
		return true
	}
	return false
}

func LinkMultipleParallel(ctx context.Context, objFiles []string, bin string, verbose bool, mode string, noSymbolCheck bool, sanitize bool, strict bool, libs []string, jobs int) error {
	if shouldSkipLinker() {
		if len(objFiles) != 1 {
			return errors.New("flat binary link requires exactly one object")
		}
		return linkFlatBinary(ctx, objFiles[0], bin)
	}
	if len(objFiles) == 0 {
		return errors.New("no object files to link")
	}
	sort.Strings(objFiles)
	if err := validateLinkInputs(ctx, objFiles, bin, noSymbolCheck, verbose); err != nil {
		return err
	}
	for _, obj := range objFiles {
		if isArchiveFile(obj) {
			if verbose {
				os.Stdout.WriteString("Archive library detected; falling back to single-stage link\n")
			}
			return linkMultipleSingle(ctx, objFiles, bin, verbose, mode, sanitize, strict, libs)
		}
	}
	if jobs <= 0 {
		jobs = 1
	}
	if len(objFiles) < minParallelLinkObjects || jobs < 2 {
		return linkMultipleSingle(ctx, objFiles, bin, verbose, mode, sanitize, strict, libs)
	}

	tmpDir, err := os.MkdirTemp(filepath.Dir(bin), ".fz_link_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	partitionCount := jobs
	if partitionCount > len(objFiles)/minPartitionSize {
		partitionCount = len(objFiles) / minPartitionSize
	}
	if partitionCount < 2 {
		return linkMultipleSingle(ctx, objFiles, bin, verbose, mode, sanitize, strict, libs)
	}
	if partitionCount > len(objFiles) {
		partitionCount = len(objFiles)
	}

	partitions := splitObjects(objFiles, partitionCount)
	targets := make([]LinkTarget, 0, len(partitions)+1)
	for i, objs := range partitions {
		partName := filepath.Join(tmpDir, "partial-"+strconv.Itoa(i)+".o")
		targets = append(targets, LinkTarget{Name: partName, Objs: objs})
	}
	finalObjs := make([]string, len(targets))
	for i, t := range targets {
		finalObjs[i] = t.Name
	}
	targets = append(targets, LinkTarget{Name: bin, Objs: finalObjs})

	dag := scheduler.NewDAGScheduler(jobs, len(targets))
	partitionIndices := make([]int, len(targets)-1)
	for i := 0; i < len(targets)-1; i++ {
		target := targets[i]
		idx, err := dag.Submit(scheduler.AcquireTask(func(arg uintptr, extra uintptr) error {
			return linkPartition(ctx, &target, mode, verbose)
		}, 0, 0), nil)
		if err != nil {
			return errors.New("failed to submit partition link task: " + err.Error())
		}
		partitionIndices[i] = idx
	}

	finalTarget := targets[len(targets)-1]
	_, err = dag.Submit(scheduler.AcquireTask(func(arg uintptr, extra uintptr) error {
		return linkFinal(ctx, &finalTarget, mode, verbose, sanitize, strict, libs)
	}, 0, 0), partitionIndices)
	if err != nil {
		return errors.New("failed to submit final link task: " + err.Error())
	}

	if err := dag.Run(ctx); err != nil {
		return err
	}

	if cfg := utils.ConfigFromContext(ctx); cfg != nil && cfg.DeterministicStrip {
		_, _ = utils.ScrubHostPaths(bin, utils.GetExecutionRoot())
	}
	return nil
}

func validateLinkInputs(ctx context.Context, objFiles []string, bin string, noSymbolCheck bool, verbose bool) error {
	for _, obj := range objFiles {
		info, err := os.Stat(obj)
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			return errors.New("object file " + obj + " is empty")
		}
	}
	if err := utils.EnsureDir(bin); err != nil {
		return err
	}
	if !noSymbolCheck {
		if err := CheckDuplicateSymbols(ctx, objFiles, verbose); err != nil {
			return err
		}
	}
	return nil
}

func linkMultipleSingle(ctx context.Context, objFiles []string, bin string, verbose bool, mode string, sanitize bool, strict bool, libs []string) error {
	if runtime.GOOS == "windows" {
		return linkWindowsImpl(ctx, objFiles, bin, verbose, sanitize, libs)
	}

	var linkErr error
	switch mode {
	case "raw":
		if linkErr = utils.CheckTool(ldForTarget()); linkErr != nil {
			return linkErr
		}
		linkErr = linkWithLd(ctx, objFiles, bin, verbose, libs)
	case "c":

		hasArchive := false
		for _, o := range objFiles {
			if isArchiveFile(o) {
				hasArchive = true
				break
			}
		}
		if hasArchive {
			if linkErr = utils.CheckTool(gccForTarget()); linkErr != nil {
				return linkErr
			}
			linkErr = linkWithGcc(ctx, objFiles, bin, verbose, false, sanitize, strict, libs)
			break
		}
		if useZig() {
			linkErr = linkWithZig(ctx, objFiles, bin, verbose, Target, sanitize, strict, libs)
			break
		}
		if linkErr = utils.CheckTool(gccForTarget()); linkErr != nil {
			return linkErr
		}
		linkErr = linkWithGcc(ctx, objFiles, bin, verbose, false, sanitize, strict, libs)
	case "auto":
		linkErr = tryAutoLink(ctx, objFiles, bin, verbose, sanitize, strict, libs)
	default:
		return errors.New("unsupported mode: " + mode + " (valid: auto, c, raw)")
	}
	return linkErr
}

func splitObjects(objFiles []string, partitions int) [][]string {
	if partitions <= 1 {
		return [][]string{objFiles}
	}
	size := (len(objFiles) + partitions - 1) / partitions
	result := make([][]string, partitions)
	for i := 0; i < partitions; i++ {
		start := i * size
		end := start + size
		if end > len(objFiles) {
			end = len(objFiles)
		}
		result[i] = objFiles[start:end]
	}
	return result
}

func linkPartition(ctx context.Context, target *LinkTarget, mode string, verbose bool) error {
	if runtime.GOOS == "windows" {
		return linkWindowsImpl(ctx, target.Objs, target.Name, verbose, false, nil)
	}
	if err := utils.CheckTool(ldForTarget()); err != nil {
		return err
	}
	return linkWithLdRelocatable(ctx, target.Objs, target.Name, verbose)
}

func linkFinal(ctx context.Context, target *LinkTarget, mode string, verbose bool, sanitize bool, strict bool, libs []string) error {
	if runtime.GOOS == "windows" {
		return linkWindowsImpl(ctx, target.Objs, target.Name, verbose, sanitize, libs)
	}
	var linkErr error
	switch mode {
	case "raw":
		if linkErr = utils.CheckTool(ldForTarget()); linkErr != nil {
			return linkErr
		}
		linkErr = linkWithLd(ctx, target.Objs, target.Name, verbose, libs)
	case "c":
		if useZig() {
			linkErr = linkWithZig(ctx, target.Objs, target.Name, verbose, Target, sanitize, strict, libs)
			break
		}
		if linkErr = utils.CheckTool(gccForTarget()); linkErr != nil {
			return linkErr
		}
		linkErr = linkWithGcc(ctx, target.Objs, target.Name, verbose, false, sanitize, strict, libs)
	case "auto":
		linkErr = tryAutoLink(ctx, target.Objs, target.Name, verbose, sanitize, strict, libs)
	default:
		return errors.New("unsupported mode: " + mode + " (valid: auto, c, raw)")
	}
	return linkErr
}

func linkWithLdRelocatable(ctx context.Context, objs []string, out string, verbose bool) error {
	if err := validateLinkCall(ctx, out); err != nil {
		return err
	}
	cmd := ldForTarget()
	args := make([]string, 0, len(objs)+4)
	args = append(args, objs...)
	args = append(args, "-r", "-o", out)
	if verbose {
		os.Stdout.WriteString("Running: " + cmd + " ")
		for i, a := range args {
			if i > 0 {
				os.Stdout.WriteString(" ")
			}
			os.Stdout.WriteString(a)
		}
		os.Stdout.WriteString("\n")
	}
	output, err := runLinkerCommand(ctx, verbose, cmd, args)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if !verbose {
			return errors.New("ld partial link failed (use -verbose for details)")
		}
		return errors.New(cmd + " failed: " + err.Error() + "\n" + output)
	}
	return nil
}
