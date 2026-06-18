package gloria

import (
	"os"
	"strconv"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func callUint64(ptr unsafe.Pointer) uint64

func TestKernelPokeAndPeek(t *testing.T) {
	src, err := os.ReadFile("test_kernel.glo")
	if err != nil {
		t.Fatal("read test_kernel.glo: " + err.Error())
	}
	if _, err = Emit(string(src)); err != nil {
		t.Fatal("compile test_kernel: " + err.Error())
	}

	pageSize := os.Getpagesize()

	fd, err := unix.MemfdCreate("forgezero", unix.MFD_CLOEXEC|unix.MFD_EXEC)
	if err != nil {
		t.Fatal("memfd_create: " + err.Error())
	}
	defer unix.Close(fd)
	if err := unix.Ftruncate(fd, int64(pageSize)); err != nil {
		t.Fatal("ftruncate: " + err.Error())
	}
	execArea, err := unix.Mmap(fd, 0, pageSize,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_SHARED)
	if err != nil {
		t.Fatal("mmap exec: " + err.Error())
	}
	defer func() {
		if err := unix.Munmap(execArea); err != nil {
			t.Fatal("failed to munmap: " + err.Error())
		}
	}()

	dataArea, err := unix.Mmap(-1, 0, pageSize,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_ANON|unix.MAP_PRIVATE|unix.MAP_32BIT)
	if err != nil {
		t.Fatal("mmap data: " + err.Error())
	}
	defer func() {
		if err := unix.Munmap(dataArea); err != nil {
			t.Fatal("failed to munmap: " + err.Error())
		}
	}()

	videoMemAddr := &dataArea[0]
	*(*uint16)(unsafe.Pointer(videoMemAddr)) = 0x0F41

	addrStr := strconv.Itoa(int(uintptr(unsafe.Pointer(videoMemAddr))))
	program := `fn main() {
    let screen = ` + addrStr + `;
    let original_char = peek(screen);
    poke(screen, 2631);
    let new_char = peek(screen);
    return new_char;
}`

	machineCode, err := Emit(program)
	if err != nil {
		t.Fatal("compile error: " + err.Error())
	}
	copy(execArea, machineCode)

	if err := unix.Mprotect(execArea, unix.PROT_READ|unix.PROT_EXEC); err != nil {
		t.Fatal("mprotect execArea: " + err.Error())
	}

	simpleCode := []byte{0x48, 0xc7, 0xc0, 0x47, 0x0a, 0x00, 0x00, 0xc3}
	simplePage, err := unix.Mmap(-1, 0, pageSize,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		t.Fatal("mmap simple: " + err.Error())
	}
	defer func() {
		if err := unix.Munmap(simplePage); err != nil {
			t.Fatal("failed to munmap: " + err.Error())
		}
	}()

	copy(simplePage, simpleCode)
	if err := unix.Mprotect(simplePage, unix.PROT_READ|unix.PROT_EXEC); err != nil {
		t.Fatal("mprotect simple: " + err.Error())
	}
	if res := callUint64(unsafe.Pointer(&simplePage[0])); res != 2631 {
		t.Fatal("simple code returned " + strconv.Itoa(int(res)))
	}

	result := callUint64(unsafe.Pointer(&execArea[0]))
	if result != 2631 {
		t.Error("expected 2631, got " + strconv.Itoa(int(result)))
	}
}