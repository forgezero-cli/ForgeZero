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

type ExitPanic struct{ Code int }

type FakeRunner struct{}

func (FakeRunner) Run(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-o" {
			out := args[i+1]
			data := []byte("BINARY")
			_ = os.WriteFile(out, data, 0o755)
			return "", nil
		}
	}
	return "", nil
}

type MuslContext struct {
	Use      bool
	Arch     string
	Target   string
	KeepObj  bool
	NoCache  bool
	BuildType string
}

type BuildContext struct {
	SrcPath       string
	DirPath       string
	OutBin        string
	OutObj        string
	Mode          string
	Debug         bool
	Verbose       bool
	KeepObj       bool
	NoCache       bool
	NoSymbolCheck bool
	Sanitize      bool
	Strict        bool
	Format        string
	Jobs          int
	BuildType     string
	JSONOutput    bool
	MuslCtx       MuslContext
}