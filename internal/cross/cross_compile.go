// Experimental! Todo after.

package cross

import (
	"context"
	"os"
	"syscall"
	"unsafe"
)

var (
	ToolchainCross   [4]byte
	CrossCompilation bool
	TargetArch       [16]byte
	TargetOS         [16]byte
)

func init() {
	ToolchainCross[0] = 'z'
	ToolchainCross[1] = 'i'
	ToolchainCross[2] = 'g'
	ToolchainCross[3] = 0
}

func writeStderr(s string) {
	b := unsafe.StringData(s)
	syscall.Write(syscall.Stderr, unsafe.Slice(b, len(s)))
}

func zigExists() bool {
	var stat syscall.Stat_t
	for _, p := range [][]byte{
		[]byte("/usr/local/bin/zig"),
		[]byte("/usr/bin/zig"),
	} {
		if syscall.Stat(string(p), &stat) == nil {
			return true
		}
	}
	return false
}

func runCommand(path string, argv []string) error {
	argvPtr := make([]uintptr, len(argv)+1)
	for i, arg := range argv {
		argPtr, err := syscall.BytePtrFromString(arg)
		if err != nil {
			return err
		}
		argvPtr[i] = uintptr(unsafe.Pointer(argPtr))
	}
	argvPtr[len(argv)] = 0

	pid, _, err := syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if err != 0 {
		return err
	}

	if pid == 0 {
		pathPtr, _ := syscall.BytePtrFromString(path)
		syscall.Syscall(syscall.SYS_EXECVE,
			uintptr(unsafe.Pointer(pathPtr)),
			uintptr(unsafe.Pointer(&argvPtr[0])),
			uintptr(unsafe.Pointer(environ)),
		)
		syscall.Exit(1)
	}

	var status syscall.WaitStatus
	syscall.Wait4(int(pid), &status, 0, nil)

	if status.ExitStatus() != 0 {
		writeStderr("failed\n")
	}
	return nil
}

func CrossCompile(ctx context.Context, src string, out string, arch string, os_ string) error {
	if !zigExists() {
		writeStderr("zig not found\n")
		return syscall.ENOENT
	}

	argv := [8]string{}
	argv[0] = "zig"
	argv[1] = "build-exe"
	argv[2] = "-target"

	target := [32]byte{}
	i := 0
	for i < len(arch) && arch[i] != 0 {
		target[i] = arch[i]
		i++
	}
	target[i] = '-'
	i++
	for j := 0; j < len(os_) && os_[j] != 0; j++ {
		target[i] = os_[j]
		i++
	}
	argv[3] = unsafe.String(&target[0], i)
	argv[4] = src

	emit := [256]byte{}
	copy(emit[:], "-femit-bin=")
	idx := 11
	for k := 0; k < len(out) && out[k] != 0; k++ {
		emit[idx] = out[k]
		idx++
	}
	argv[5] = unsafe.String(&emit[0], idx)
	argv[6] = "-O"
	argv[7] = "ReleaseFast"

	return runCommand("/usr/local/bin/zig", []string{argv[0], argv[1], argv[2], argv[3], argv[4], argv[5], argv[6], argv[7]})
}

var environ uintptr

func init() {
	env := os.Environ()
	envPtrs := make([]uintptr, len(env)+1)
	for i, e := range env {
		ptr, _ := syscall.BytePtrFromString(e)
		envPtrs[i] = uintptr(unsafe.Pointer(ptr))
	}
	environ = uintptr(unsafe.Pointer(&envPtrs[0]))
}