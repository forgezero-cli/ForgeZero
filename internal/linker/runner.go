package linker

import (
	"context"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
)

type CmdRunner interface {
	Run(ctx context.Context, verbose bool, name string, args ...string) (output string, err error)
}

type RealCmdRunner struct{}

func (r *RealCmdRunner) Run(ctx context.Context, verbose bool, name string, args ...string) (string, error) {
	return utils.RunCommandSilent(ctx, verbose, name, args...)
}
