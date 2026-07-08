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

package fzerr

import "testing"

func TestErrorCode(t *testing.T) {
	e := New(CodeFileNotFound)
	if e.Code != CodeFileNotFound {
		t.Fatalf("code mismatch: got %d want %d", e.Code, CodeFileNotFound)
	}
	if e.Error() != "file_not_found" {
		t.Fatalf("unexpected message: %q", e.Error())
	}
}

func TestGetCode(t *testing.T) {
	if GetCode(New(CodeParseFailed)) != CodeParseFailed {
		t.Fatal("GetCode failed for typed error")
	}
	if GetCode(nil) != CodeOK {
		t.Fatal("GetCode(nil) should be OK")
	}
}

func TestFormatError(t *testing.T) {
	msg := FormatError(CodeBuildActionFailed, "builder", "main.c", 42, "compile failed")
	if msg == "" {
		t.Fatal("empty formatted message")
	}
}

func TestAppendMsg(t *testing.T) {
	buf := AppendMsg(nil, CodeHashOpen, "path", "denied")
	if len(buf) == 0 {
		t.Fatal("empty buffer")
	}
}
