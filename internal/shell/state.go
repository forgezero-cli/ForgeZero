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

package shell

type BuildState struct {
	Mode          string
	Format        string
	Strict        bool
	Sanitize      bool
	Verbose       bool
	Debug         bool
	NoCache       bool
	NoSymbolCheck bool
	KeepObj       bool
	LdScript      string
	TextAddr      string
	Out           string
	SourcePath    string
	SourceType    string
}

func DefaultState() *BuildState {
	return &BuildState{
		Mode:          "auto",
		Format:        "elf64",
		Strict:        false,
		Sanitize:      true,
		Verbose:       false,
		Debug:         false,
		NoCache:       false,
		NoSymbolCheck: false,
		KeepObj:       false,
		LdScript:      "",
		TextAddr:      "",
		Out:           "",
		SourcePath:    ".",
		SourceType:    "dir",
	}
}
