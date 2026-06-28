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

package scripts

import (
	"context"
	"os"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

var runCommand = utils.RunCommand

type ScriptsConfigure struct {
	Commands []string
	Verbose  bool
}

func (s *ScriptsConfigure) Run(ctx context.Context) error {
	for _, cmd := range s.Commands {
		_, err := runCommand(ctx, false, os.Stdout, os.Stderr, "sh", "-c", cmd)
		if err != nil {
			return err
		}
	}
	return nil
}
