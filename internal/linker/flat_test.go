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

package linker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/forgezero-cli/ForgeZero/internal/assembler"
)

func TestLinkFlatBinaryCopy(t *testing.T) {
	old := assembler.OutputFormat
	defer func() { assembler.OutputFormat = old }()
	assembler.OutputFormat = "bin"
	dir := t.TempDir()
	obj := filepath.Join(dir, "flat.bin")
	payload := []byte{0x90, 0x90, 0xeb, 0xfe}
	if err := os.WriteFile(obj, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.bin")
	if err := Link(context.Background(), obj, out, false, "raw", true, false, false, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(payload) {
		t.Fatalf("len %d want %d", len(got), len(payload))
	}
	for i := range payload {
		if got[i] != payload[i] {
			t.Fatalf("byte %d mismatch", i)
		}
	}
}

func TestLinkFlatBinaryNoCopy(t *testing.T) {
	old := assembler.OutputFormat
	defer func() { assembler.OutputFormat = old }()
	assembler.OutputFormat = "bin"
	dir := t.TempDir()
	obj := filepath.Join(dir, "same.bin")
	if err := os.WriteFile(obj, []byte{0xcd}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Link(context.Background(), obj, obj, false, "raw", true, false, false, nil); err != nil {
		t.Fatal(err)
	}
}
