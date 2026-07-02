/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package core

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/forgezero-cli/ForgeZero/internal/iso/hybrid"
)

var (
	isoToolPath string
	isoToolErr  error
	isoToolOnce sync.Once
	lookPath    = exec.LookPath
	command     = exec.Command
)

var toolCandidates = [...]string{"xorriso", "genisoimage", "mkisofs"}

type Options struct {
	SourceDir     string
	OutputPath    string
	VolumeID      string
	BootCatalog   string
	BootImage     string
	BootLoadSize  string
	NoEmulBoot    bool
	BootInfoTable bool
	Joliet        bool
	RockRidge     bool
	Hybrid        bool
	CustomArgs    []string
}

func Discover() (string, error) {
	isoToolOnce.Do(func() {
		for _, name := range toolCandidates {
			if path, err := lookPath(name); err == nil {
				isoToolPath = path
				return
			}
		}
		isoToolErr = errors.New("no ISO creation tool found (xorriso, genisoimage, mkisofs)")
	})
	return isoToolPath, isoToolErr
}

func Build(opts Options) error {
	if opts.SourceDir == "" {
		return errors.New("source directory cannot be empty")
	}
	info, err := os.Stat(opts.SourceDir)
	if err != nil {
		return errors.New("failed to access source directory " + opts.SourceDir + ": " + err.Error())
	}
	if !info.IsDir() {
		return errors.New("source path is not a directory: " + opts.SourceDir)
	}
	if opts.OutputPath == "" {
		opts.OutputPath = "output.iso"
	}
	if !strings.HasSuffix(opts.OutputPath, ".iso") {
		opts.OutputPath += ".iso"
	}
	cmdPath, err := Discover()
	if err != nil {
		return err
	}
	args := make([]string, 0, 32)
	args = append(args, "-o", opts.OutputPath)
	if opts.VolumeID != "" {
		args = append(args, "-V", opts.VolumeID)
	}
	if opts.BootImage != "" {
		args = append(args, "-b", opts.BootImage)
		if opts.BootCatalog != "" {
			args = append(args, "-c", opts.BootCatalog)
		}
		if opts.NoEmulBoot {
			args = append(args, "-no-emul-boot")
		}
		if opts.BootLoadSize != "" {
			args = append(args, "-boot-load-size", opts.BootLoadSize)
		}
		if opts.BootInfoTable {
			args = append(args, "-boot-info-table")
		}
	}
	if opts.Joliet {
		args = append(args, "-J", "-R")
	}
	if opts.RockRidge && !opts.Joliet {
		args = append(args, "-R")
	}
	if opts.Hybrid {
		args = append(args, hybrid.Args(cmdPath)...)
	}
	if len(opts.CustomArgs) > 0 {
		args = append(args, opts.CustomArgs...)
	}
	args = append(args, opts.SourceDir)
	cmd := command(cmdPath, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return errors.New("ISO creation failed (" + cmdPath + "): " + err.Error())
	}
	return nil
}
