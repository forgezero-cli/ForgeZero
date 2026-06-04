package gloria

import (
	"fmt"
	"os"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func callUint64(ptr unsafe.Pointer) uint64

func TestKernelPokeAndPeek(t *testing.T) {
	src, err := os.ReadFile("test_kernel.glo")
	if err != nil {
		t.Fatalf("read test_kernel.glo: %v", err)
	}
	if _, err = Emit(string(src)); err != nil {
		t.Fatalf("compile test_kernel: %v", err)
	}

	pageSize := os.Getpagesize()

	fd, err := unix.MemfdCreate("forgezero", unix.MFD_CLOEXEC|unix.MFD_EXEC)
	if err != nil {
		t.Fatalf("memfd_create: %v", err)
	}
	defer unix.Close(fd)
	if err := unix.Ftruncate(fd, int64(pageSize)); err != nil {
		t.Fatalf("ftruncate: %v", err)
	}
	execArea, err := unix.Mmap(fd, 0, pageSize,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_SHARED)
	if err != nil {
		t.Fatalf("mmap exec: %v", err)
	}

	defer func() {
		if err := unix.Munmap(execArea); err != nil {
			t.Fatalf("failed to mummap: %v", err)
		}
	}()

	dataArea, err := unix.Mmap(-1, 0, pageSize,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_ANON|unix.MAP_PRIVATE|unix.MAP_32BIT)
	if err != nil {
		t.Fatalf("mmap data: %v", err)
	}

	defer func() {
		if err := unix.Munmap(dataArea); err != nil {
			t.Fatalf("failed to munmap: %v", err)
		}
	}()

	videoMemAddr := uintptr(unsafe.Pointer(&dataArea[0]))
	*(*uint16)(unsafe.Pointer(videoMemAddr)) = 0x0F41
	t.Logf("videoMemAddr = %d (dec) = %#x (hex)", videoMemAddr, videoMemAddr)

	testProgram := fmt.Sprintf(`
fn main() {
    let screen = %d;
    let original_char = peek(screen);
    poke(screen, 2631);
    let new_char = peek(screen);
    return new_char;
}
`, videoMemAddr)

	machineCode, err := Emit(testProgram)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	copy(execArea, machineCode)

	if err := unix.Mprotect(execArea, unix.PROT_READ|unix.PROT_EXEC); err != nil {
		t.Fatalf("mprotect execArea: %v", err)
	}

	simpleCode := []byte{0x48, 0xc7, 0xc0, 0x47, 0x0a, 0x00, 0x00, 0xc3}
	simplePage, err := unix.Mmap(-1, 0, pageSize,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		t.Fatalf("mmap simple: %v", err)
	}

	defer func() {
		if err := unix.Munmap(simplePage); err != nil {
			t.Fatalf("failed to munmap: %v", err)
		}
	}()

	copy(simplePage, simpleCode)
	if err := unix.Mprotect(simplePage, unix.PROT_READ|unix.PROT_EXEC); err != nil {
		t.Fatalf("mprotect simple: %v", err)
	}
	if res := callUint64(unsafe.Pointer(&simplePage[0])); res != 2631 {
		t.Fatalf("simple code returned %d", res)
	}

	result := callUint64(unsafe.Pointer(&execArea[0]))
	if result != 2631 {
		t.Errorf("expected 2631, got %d", result)
	}
}
