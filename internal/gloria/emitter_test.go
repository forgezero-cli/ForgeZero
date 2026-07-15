//go:build linux
// +build linux

/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package gloria

import (
	"bytes"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"syscall"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/utils"
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

	err = unix.Dup2(int(w.Fd()), stdoutFd)
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
	_ = unix.Dup2(clonedStdoutFd, stdoutFd)
	_ = syscall.Close(clonedStdoutFd)

	result := <-outChan
	expected := "hello world"

	if result != expected {
		t.Errorf("unexpected result: got %q, want %q", result, expected)
	}
}
