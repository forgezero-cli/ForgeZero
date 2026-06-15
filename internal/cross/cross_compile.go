// Experimental! Todo after.

package cross

import (
	"context"
	"os"
	"syscall"
	"unsafe"
)

func writeStderr(s string) {
	b := unsafe.StringData(s)
	syscall.Write(syscall.Stderr, unsafe.Slice(b, len(s)))
}

func forkSyscall() uintptr {
	if syscall.SYS_FORK != 0 {
		return syscall.SYS_FORK
	}
	return syscall.SYS_CLONE
}

func runCommand(path string, argv []string, envv []string) error {
	argvPtrs := make([]uintptr, len(argv)+1)
	for i, arg := range argv {
		ptr, err := syscall.BytePtrFromString(arg)
		if err != nil {
			return err
		}
		argvPtrs[i] = uintptr(unsafe.Pointer(ptr))
	}
	argvPtrs[len(argv)] = 0

	envPtrs := make([]uintptr, len(envv)+1)
	for i, env := range envv {
		ptr, err := syscall.BytePtrFromString(env)
		if err != nil {
			return err
		}
		envPtrs[i] = uintptr(unsafe.Pointer(ptr))
	}
	envPtrs[len(envv)] = 0

	forkNum := forkSyscall()
	if forkNum == 0 {
		writeStderr("no fork/clone\n")
		return syscall.ENOSYS
	}

	pid, _, err := syscall.Syscall(forkNum, 0, 0, 0)
	if err != 0 {
		return err
	}

	if pid == 0 {
		pathPtr, _ := syscall.BytePtrFromString(path)
		syscall.Syscall(syscall.SYS_EXECVE,
			uintptr(unsafe.Pointer(pathPtr)),
			uintptr(unsafe.Pointer(&argvPtrs[0])),
			uintptr(unsafe.Pointer(&envPtrs[0])),
		)
		syscall.Exit(1)
	}

	var status syscall.WaitStatus
	syscall.Wait4(int(pid), &status, 0, nil)
	if status.ExitStatus() != 0 {
		writeStderr("fail\n")
		return syscall.EINVAL
	}
	return nil
}

func CrossCompile(ctx context.Context, src, out, arch, os_ string) error {
	writeStderr("cross: ")
	writeStderr(src)
	writeStderr(" -> ")
	writeStderr(arch)
	writeStderr("-")
	writeStderr(os_)
	writeStderr("\n")

	return runCommand("/usr/local/bin/zig", []string{
		"zig", "build-exe", "-target", arch + "-" + os_,
		src, "-femit-bin=" + out, "-O", "ReleaseFast",
	}, os.Environ())
}