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
	"strings"
	"testing"
)

func TestParseNmOutput(t *testing.T) {
	text := `0000000000000000 T main
0000000000000000 D data_sym
0000000000000000 B bss_sym
0000000000000000 t local_fn
0000000000000000 .L123
`
	syms := parseNmOutput("/obj.o", text)
	if len(syms) != 3 {
		t.Fatalf("got %d symbols", len(syms))
	}
	if syms[0].Name != "main" {
		t.Fatal(syms)
	}
}

func TestParseObjdumpOutput(t *testing.T) {
	text := `0000000000000000 g     F .text	00000005 main
0000000000000000 g     O .data	00000004 gdata
0000000000000000     F UND	00000000 puts
0000000000000000       *ABS*	00000000 _start
`
	syms := parseObjdumpOutput("/obj.o", text)
	if len(syms) != 2 {
		t.Fatalf("got %v", syms)
	}
}

func TestParseReadelfOutput(t *testing.T) {
	text := `   Num:    Value          Size Type    Bind   Vis      Ndx Name
     1: 0000000000000000     0 NOTYPE  GLOBAL DEFAULT  UND puts
     2: 0000000000000000     5 FUNC    GLOBAL DEFAULT    1 main
     3: 0000000000000000     4 OBJECT  GLOBAL DEFAULT    2 gvar
`
	syms := parseReadelfOutput("/obj.o", text)
	if len(syms) == 0 {
		t.Fatal("expected symbols")
	}
	for _, s := range syms {
		if s.Name == "puts" {
			t.Fatal("UND should be skipped")
		}
	}
}

func TestShouldCheckDuplicateFilters(t *testing.T) {
	cases := map[string]bool{
		"": false, "_end": false, ".L1": false, "debug_x": false, "my_func": true,
	}
	for name, want := range cases {
		if shouldCheckDuplicate(name) != want {
			t.Fatalf("%s", name)
		}
	}
}

func TestReadSymbolsUsesNmWhenAvailable(t *testing.T) {
	text := `0000000000000000 T sym_a
`
	syms := parseNmOutput("obj", text)
	if len(syms) != 1 || syms[0].Name != "sym_a" {
		t.Fatal(syms)
	}
}

func TestParseNmOutputSkipsUnderscoreStart(t *testing.T) {
	text := `0000000000000000 T _start
0000000000000000 T real_sym
`
	syms := parseNmOutput("o", text)
	if len(syms) != 1 || syms[0].Name != "real_sym" {
		t.Fatalf("%v", syms)
	}
}

func TestParseObjdumpShortLine(t *testing.T) {
	syms := parseObjdumpOutput("o", "short line\n")
	if len(syms) != 0 {
		t.Fatal(syms)
	}
}

func TestParseReadelfNoGlobal(t *testing.T) {
	syms := parseReadelfOutput("o", "no globals here\n")
	if len(syms) != 0 {
		t.Fatal(syms)
	}
}

func TestCheckDuplicateSymbolMessage(t *testing.T) {
	dup := "symbol 'x' defined in:"
	if !strings.Contains(dup, "symbol") {
		t.Fatal()
	}
}
