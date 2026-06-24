// SPDX-License-Identifier: MIT
// Author: Alex Voste

package scripts

import (
	"context"
	"fz/internal/utils"
	"os"
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
