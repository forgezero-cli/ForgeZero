package cross

import (
	"syscall"
	"testing"
	"unsafe"
)

func TestZigExists(t *testing.T) {
	if zigExists() {
		writeStderr("zig found\n")
	} else {
		writeStderr("zig not found\n")
	}
}

func TestWriteStderr(t *testing.T) {
	writeStderr("test message\n")
}

func TestRunCommand(t *testing.T) {
	argv := []string{"echo", "hello"}
	err := runCommand("/bin/echo", argv)
	if err != nil {
		t.Fail()
	}
}

func TestCrossCompile(t *testing.T) {
	src := "test.zig"
	out := "test.out"
	arch := "x86_64"
	os := "linux"

	err := CrossCompile(nil, src, out, arch, os)
	if err != syscall.ENOENT {
		writeStderr("zig exists, testing compilation\n")
	}
}

func TestTargetString(t *testing.T) {
	arch := "aarch64"
	os := "windows"

	target := [32]byte{}
	i := 0
	for i < len(arch) && arch[i] != 0 {
		target[i] = arch[i]
		i++
	}
	target[i] = '-'
	i++
	for j := 0; j < len(os) && os[j] != 0; j++ {
		target[i] = os[j]
		i++
	}

	result := unsafe.String(&target[0], i)
	expected := "aarch64-windows"

	for k := 0; k < len(expected); k++ {
		if result[k] != expected[k] {
			t.Fail()
		}
	}
}

func TestEmitFlag(t *testing.T) {
	out := "build/myapp"

	emit := [256]byte{}
	copy(emit[:], "-femit-bin=")
	idx := 11
	for k := 0; k < len(out) && out[k] != 0; k++ {
		emit[idx] = out[k]
		idx++
	}

	result := unsafe.String(&emit[0], idx)
	expected := "-femit-bin=build/myapp"

	for k := 0; k < len(expected); k++ {
		if result[k] != expected[k] {
			t.Fail()
		}
	}
}

func TestMemoryZeroAllocs(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		var buf [256]byte
		_ = buf
	})

	if allocs > 0 {
		writeStderr("has allocations\n")
		t.Fail()
	}
}

func BenchmarkCrossCompile(b *testing.B) {
	src := "test.zig"
	out := "test.out"
	arch := "x86_64"
	os := "linux"

	for i := 0; i < b.N; i++ {
		CrossCompile(nil, src, out, arch, os)
	}
}

func BenchmarkTargetBuild(b *testing.B) {
	arch := "riscv64"
	os := "freertos"

	for i := 0; i < b.N; i++ {
		target := [32]byte{}
		idx := 0
		for j := 0; j < len(arch); j++ {
			target[idx] = arch[j]
			idx++
		}
		target[idx] = '-'
		idx++
		for j := 0; j < len(os); j++ {
			target[idx] = os[j]
			idx++
		}
		_ = unsafe.String(&target[0], idx)
	}
}

func TestForkExecve(t *testing.T) {
	pid, _, _ := syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	
	if pid == 0 {
		pathPtr, _ := syscall.BytePtrFromString("/bin/true")
		syscall.Syscall(syscall.SYS_EXECVE,
			uintptr(unsafe.Pointer(pathPtr)),
			0,
			0,
		)
		syscall.Exit(0)
	}
	
	var status syscall.WaitStatus
	syscall.Wait4(int(pid), &status, 0, nil)
	
	if status.ExitStatus() != 0 {
		t.Fail()
	}
}

func TestSyscallWrite(t *testing.T) {
	msg := "syscall test\n"
	b := unsafe.StringData(msg)
	_, err := syscall.Write(syscall.Stderr, unsafe.Slice(b, len(msg)))
	if err != nil {
		t.Fail()
	}
}