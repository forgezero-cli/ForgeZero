package assembler

import "fz/internal/utils"

func ensureAsmTool(name string) error {
	switch name {
	case "nasm", "fasm":
		return CheckAssemblerTool(name)
	default:
		return utils.CheckTool(name)
	}
}

func nasmExecutable() (string, error) {
	return getToolPath("nasm")
}

func fasmExecutable() (string, error) {
	return getToolPath("fasm")
}

func assembleNASMSlowCmd() (string, error) {
	cmd := asmCmdForTarget()
	switch cmd {
	case "nasm":
		return nasmExecutable()
	case "fasm":
		return fasmExecutable()
	default:
		if err := utils.CheckTool(cmd); err != nil {
			return "", err
		}
		return cmd, nil
	}
}
