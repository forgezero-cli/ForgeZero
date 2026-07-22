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

import (
	"strings"
	"testing"
)

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

func TestRenderLineError(t *testing.T) {
	file := []byte("[fz]\ncompiler = \"gcc\"\nsources_dirs = [\"src\", \"lib\"]\n")
	out := RenderLineError(file, 3, "sources_dirs", "the configured directory does not exist", "ensure the path exists")
	if len(out) == 0 {
		t.Fatal("empty rendered diagnostic")
	}
	text := string(out)
	if !strings.Contains(text, "sources_dirs") {
		t.Fatalf("expected diagnostic to mention parameter, got %q", text)
	}
	if !strings.Contains(text, "ensure the path exists") {
		t.Fatalf("expected diagnostic to include hint, got %q", text)
	}
}
