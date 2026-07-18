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

package stdio

import (
	"encoding/json"
	"os"
)

type BuildReport struct {
	Status      string   `json:"status"`
	ExitCode    int      `json:"exit_code"`
	DurationMs  int64    `json:"duration_ms"`
	Binary      string   `json:"binary,omitempty"`
	SourceFiles []string `json:"source_files,omitempty"`
	ObjectFiles []string `json:"object_files,omitempty"`
	Error       string   `json:"error,omitempty"`
}

func EncodeBuildReport(report BuildReport) error {
	return json.NewEncoder(os.Stdout).Encode(report)
}

func WriteErrorReport(exitCode int, err error) {
	if e := EncodeBuildReport(BuildReport{
		Status:     "error",
		ExitCode:   exitCode,
		DurationMs: 0,
		Error:      err.Error(),
	}); e != nil {
		_, _ = os.Stderr.WriteString("failed to write error report: ")
		_, _ = os.Stderr.WriteString(e.Error())
		_, _ = os.Stderr.WriteString("\n")
	}
}

func WriteSuccessReport(durationMs int64, binary string, sourceFiles, objectFiles []string) error {
	return EncodeBuildReport(BuildReport{
		Status:      "success",
		ExitCode:    0,
		DurationMs:  durationMs,
		Binary:      binary,
		SourceFiles: sourceFiles,
		ObjectFiles: objectFiles,
	})
}
