package gloria

import (
	"bytes"
	"io"
	"os"
	"syscall"
	"testing"

	"fz/internal/utils"
)

func TestGloriaRegisterCarnage(t *testing.T) {
	src := `

	fn main() {
    let x = 10
    let y = 5

    print("hello")
    return add(x, y)
}

fn add(a, b) {
    return a + b
}



`

	code, err := Emit(src)
	if err != nil {
		t.Fatalf("Compiler error: %v", err)
	}

	out := utils.ExecRawRet(code)

	result := int(out)
	expected := 15
	if result != expected {

		t.Errorf("Unexpected result: got %d, want %d", result, expected)
	}

	t.Logf("Gloria Register Carnage success! Result: %d, Machine code size: %d bytes", out, len(code))
}

func TestGloriaWhile(t *testing.T) {
	src := `
	fn main() {
		let running = 3
		while running {
			print("x")
			running -= 1
		}
		return running
	}
	`

	code, err := Emit(src)
	if err != nil {
		t.Fatalf("compiler err: %v", err)
	}

	out := utils.ExecRawRet(code)
	result := int(out)
	expected := 0
	if result != expected {
		t.Errorf("unexpected result: got %d, want %d", result, expected)
	}
}

func TestGloriaMath(t *testing.T) {

	src := `
	fn main() {
		let a = 150
		let b = 20

		return a - b
	}
	`

	code, err := Emit(src)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	out := utils.ExecRawRet(code)

	result := int(out)

	expected := 130

	if result != expected {
		t.Errorf("unexpected result: got %d, want %d", result, expected)
	}

	t.Logf("gloria test math success! result: %d. Machine code size: %d bytes", out, len(code))
}

func TestGloriaPrint(t *testing.T) {
	src := `
	fn main() {
		print("hello world")
	}
	`

	code, err := Emit(src)
	if err != nil {
		t.Fatalf("compiler err: %v", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	stdoutFd := int(os.Stdout.Fd())
	clonedStdoutFd, err := syscall.Dup(stdoutFd)
	if err != nil {
		t.Fatalf("failed to clone stdout fd: %v", err)
	}

	err = syscall.Dup2(int(w.Fd()), stdoutFd)
	if err != nil {
		t.Fatalf("failed to redirect stdout fd: %v", err)
	}

	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	utils.ExecRawRet(code)

	w.Close()
	_ = syscall.Dup2(clonedStdoutFd, stdoutFd)
	_ = syscall.Close(clonedStdoutFd)

	result := <-outChan
	expected := "hello world"

	if result != expected {
		t.Errorf("unexpected result: got %q, want %q", result, expected)
	}
}
