//go:build !unix && !windows

package assembler

import "os"

func materializeEmbedded(data []byte, tool string) (string, error) {
	_ = data
	_ = tool
	return "", os.ErrNotExist
}

func rejectPathLen() {
	writeErrHot(errAsmPath)
	os.Exit(errAsmPath)
}
