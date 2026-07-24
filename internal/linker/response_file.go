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
	"bufio"
	"errors"
	"os"
	"strings"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

const (
	maxResponseFileBytes = 32 * 1024
	maxResponseFileArg   = 8192
)

func shouldUseResponseFile(args []string) bool {
	if len(args) > 128 {
		return true
	}
	total := 0
	for _, arg := range args {
		if len(arg) > maxResponseFileArg {
			return true
		}
		total += len(arg) + 1
		if total > maxResponseFileBytes {
			return true
		}
	}
	return false
}

func cleanupResponseFile(f *os.File, name string) {
	if f != nil {
		_ = f.Close()
	}
	_ = os.Remove(name)
}

func createResponseFile(args []string) (string, error) {
	f, err := os.CreateTemp("", "fz_link_args_*.rsp")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Chmod(utils.FilePerm); err != nil {
		cleanupResponseFile(f, name)
		return "", err
	}
	writer := bufio.NewWriterSize(f, 64*1024)
	for _, arg := range args {
		if strings.ContainsAny(arg, "\n\r\x00") {
			cleanupResponseFile(f, name)
			return "", errors.New("invalid argument for response file")
		}
		if err := utils.ValidateCLIArg(arg); err != nil {
			cleanupResponseFile(f, name)
			return "", errors.New("invalid argument for response file: " + err.Error())
		}
		if _, err := writer.WriteString(arg); err != nil {
			cleanupResponseFile(f, name)
			return "", err
		}
		if err := writer.WriteByte('\n'); err != nil {
			cleanupResponseFile(f, name)
			return "", err
		}
	}
	if err := writer.Flush(); err != nil {
		cleanupResponseFile(f, name)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}
