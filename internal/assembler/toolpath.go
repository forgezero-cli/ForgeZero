package assembler

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

const maxToolPathLen = 4096

var toolPathInit sync.Mutex

var cachedToolPaths struct {
	nasm      [maxToolPathLen]byte
	nasmN     int
	fasm      [maxToolPathLen]byte
	fasmN     int
	nasmReady bool
	fasmReady bool
}

func validatePathLen(s string) bool {
	return len(s) <= maxToolPathLen
}

func rejectPathLen() {
	writeErrHot(errAsmPath)
	if flag.Lookup("test.v") != nil {
		panic("path length exceeded")
	}
	syscall.Exit(errAsmPath)
}

func storeToolPath(dst []byte, dstN *int, path string) bool {
	if len(path) > maxToolPathLen {
		return false
	}
	n := copy(dst[:], path)
	*dstN = n
	return true
}

func lookPathTool(name string) (string, bool) {
	if len(name) > 64 {
		return "", false
	}
	p, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	if !validatePathLen(p) {
		rejectPathLen()
	}
	return p, true
}

func resolveToolLocked(tool string) (string, error) {
	switch tool {
	case "nasm":
		if cachedToolPaths.nasmReady {
			return unsafe.String(&cachedToolPaths.nasm[0], cachedToolPaths.nasmN), nil
		}
		if p, ok := lookPathTool("nasm"); ok {
			if !storeToolPath(cachedToolPaths.nasm[:], &cachedToolPaths.nasmN, p) {
				rejectPathLen()
			}
			cachedToolPaths.nasmReady = true
			return p, nil
		}
		p, err := extractEmbeddedTool("nasm")
		if err != nil {
			return "", err
		}
		if !storeToolPath(cachedToolPaths.nasm[:], &cachedToolPaths.nasmN, p) {
			rejectPathLen()
		}
		cachedToolPaths.nasmReady = true
		return p, nil
	case "fasm":
		if cachedToolPaths.fasmReady {
			return unsafe.String(&cachedToolPaths.fasm[0], cachedToolPaths.fasmN), nil
		}
		if p, ok := lookPathTool("fasm"); ok {
			if !storeToolPath(cachedToolPaths.fasm[:], &cachedToolPaths.fasmN, p) {
				rejectPathLen()
			}
			cachedToolPaths.fasmReady = true
			return p, nil
		}
		p, err := extractEmbeddedTool("fasm")
		if err != nil {
			return "", err
		}
		if !storeToolPath(cachedToolPaths.fasm[:], &cachedToolPaths.fasmN, p) {
			rejectPathLen()
		}
		cachedToolPaths.fasmReady = true
		return p, nil
	default:
		if p, ok := lookPathTool(tool); ok {
			return p, nil
		}
		return "", os.ErrNotExist
	}
}

func getToolPath(tool string) (string, error) {
	if !validatePathLen(tool) {
		rejectPathLen()
	}
	toolPathInit.Lock()
	defer toolPathInit.Unlock()
	return resolveToolLocked(tool)
}

func getToolPathHot(tool string, out *pathBuf) bool {
	if !validatePathLen(tool) {
		return false
	}
	toolPathInit.Lock()
	defer toolPathInit.Unlock()
	p, err := resolveToolLocked(tool)
	if err != nil {
		return false
	}
	if len(p) > maxToolPathLen {
		return false
	}
	out.reset()
	copy(out.data[:], p)
	out.n = len(p)
	return true
}

func embeddedAssetName(tool string) string {
	var b [64]byte
	n := 0
	for i := 0; i < len(tool) && n < len(b)-20; i++ {
		b[n] = tool[i]
		n++
	}
	under := []byte{'_'}
	copy(b[n:], under)
	n += 1
	osName := runtime.GOOS
	for i := 0; i < len(osName) && n < len(b)-12; i++ {
		b[n] = osName[i]
		n++
	}
	copy(b[n:], under)
	n += 1
	arch := runtime.GOARCH
	for i := 0; i < len(arch) && n < len(b)-5; i++ {
		b[n] = arch[i]
		n++
	}
	if runtime.GOOS == "windows" {
		ext := []byte{'.', 'e', 'x', 'e'}
		copy(b[n:], ext)
		n += len(ext)
	}
	return unsafe.String(&b[0], n)
}

func extractEmbeddedTool(tool string) (string, error) {
	data, ok := loadEmbeddedTool(tool)
	if !ok || len(data) < 64 {
		return "", os.ErrNotExist
	}
	return materializeEmbedded(data, tool)
}

func CheckAssemblerTool(name string) error {
	_, err := getToolPath(name)
	return err
}

func ResolveNASMPath() (string, error) {
	return getToolPath("nasm")
}

func ResolveFASMPath() (string, error) {
	return getToolPath("fasm")
}

func ToolSearchDirs() []string {
	root := filepath.Clean(".")
	return []string{
		filepath.Join(root, "toolchain", "bin"),
		filepath.Join(root, "bin"),
	}
}
