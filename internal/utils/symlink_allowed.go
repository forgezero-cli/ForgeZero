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

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func symlinkAllowed(rootEval, path, target string) (bool, error) {
	linkTarget, err := fileSystem().Readlink(path)
	if err != nil {
		return false, fmt.Errorf("cannot read symlink %s: %w", path, err)
	}
	var targetAbs string
	if !filepath.IsAbs(linkTarget) {
		targetAbs = filepath.Clean(filepath.Join(filepath.Dir(path), linkTarget))
	} else {
		targetAbs = filepath.Clean(linkTarget)
	}
	targetEval, err := fileSystem().EvalSymlinks(targetAbs)
	if err != nil {
		return false, fmt.Errorf("cannot resolve symlink %s target %s: %w", path, targetAbs, err)
	}
	rootClean := filepath.Clean(rootEval)
	if targetEval == rootClean || strings.HasPrefix(targetEval, rootClean+string(os.PathSeparator)) {
		return true, nil
	}
	fmt.Fprintf(os.Stderr, "SECURITY WARNING: skipping symlink %s -> %s outside project root %s\n", path, targetAbs, rootClean)
	return false, nil
}
