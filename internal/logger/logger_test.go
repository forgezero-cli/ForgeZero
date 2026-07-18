/*
 *   Copyright (c) 2026 ForgeZero-cli

 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version of the License.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package logger

import (
	"io"
	"os"
	"testing"
)

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	_ = w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old
	return string(out)
}

func TestDebugDisabled(t *testing.T) {
	Enabled = false
	out := captureStdout(func() {
		Debug("nope")
	})
	if out != "" {
		t.Fatalf("expected empty, got %q", out)
	}
}

func TestDebugEnabled(t *testing.T) {
	Enabled = true
	out := captureStdout(func() {
		Debug("yes")
	})
	if out != "yes" {
		t.Fatalf("unexpected output %q", out)
	}
}

func TestDebugf(t *testing.T) {
	Enabled = true
	out := captureStdout(func() {
		Debugf("a %v", "b")
	})
	if out == "" {
		t.Fatalf("expected non-empty output")
	}
}

func TestToString(t *testing.T) {
	if toString("x") != "x" {
		t.Fatal("string conversion failed")
	}
	if toString([]byte("y")) != "y" {
		t.Fatal("byte conversion failed")
	}
}
