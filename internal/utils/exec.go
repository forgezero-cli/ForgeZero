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
	"os/exec"
	"runtime"
	"strings"
)

func LookExecutable(name string) (string, error) {
	return lookExecutable(name)
}

func lookExecutable(name string) (string, error) {
	if runtime.GOOS != "windows" {
		return exec.LookPath(name)
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".bat") || strings.HasSuffix(lower, ".cmd") {
		return exec.LookPath(name)
	}
	if p, err := exec.LookPath(name + ".exe"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath(name + ".bat"); err == nil {
		return p, nil
	}
	return exec.LookPath(name)
}
