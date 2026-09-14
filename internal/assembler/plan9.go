package assembler

import "path/filepath"

func plan9IncludeDirs(goroot string) (string, string) {
	return filepath.Join(goroot, "pkg", "include"), filepath.Join(goroot, "src", "runtime")
}
