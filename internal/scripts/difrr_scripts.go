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
		if s.Verbose {
			os.Stdout.WriteString("Running script: ")
			os.Stdout.WriteString(cmd)
			os.Stdout.WriteString("\n")
		}
		_, err := runCommand(ctx, s.Verbose, os.Stdout, os.Stderr, "sh", "-c", cmd)
		if err != nil {
			return err
		}
	}
	return nil
}
