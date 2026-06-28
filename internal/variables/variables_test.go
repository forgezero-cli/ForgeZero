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

package variables

import (
	"testing"
)

func TestExpandString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		vars map[string]string
		want string
	}{
		{"empty", "", nil, ""},
		{"no vars", "hello", map[string]string{}, "hello"},
		{"simple $VAR", "$FOO", map[string]string{"FOO": "bar"}, "bar"},
		{"simple ${VAR}", "${FOO}", map[string]string{"FOO": "bar"}, "bar"},
		{"missing $VAR", "$FOO", map[string]string{"BAR": "baz"}, "$FOO"},
		{"missing ${VAR}", "${FOO}", map[string]string{"BAR": "baz"}, "${FOO}"},
		{"mixed text", "hello $USER", map[string]string{"USER": "world"}, "hello world"},
		{"braced in text", "hello ${USER}", map[string]string{"USER": "world"}, "hello world"},
		{"multiple", "$A $B", map[string]string{"A": "1", "B": "2"}, "1 2"},
		{"multiple braced", "${A} ${B}", map[string]string{"A": "1", "B": "2"}, "1 2"},
		{"recursive simple", "$A", map[string]string{"A": "$B", "B": "C"}, "C"},
		{"recursive braced", "${A}", map[string]string{"A": "${B}", "B": "C"}, "C"},
		{"recursive deep", "$A", map[string]string{"A": "$B", "B": "$C", "C": "D"}, "D"},
		{"cycle protection", "$A", map[string]string{"A": "$B", "B": "$A"}, "$A"},
		{"non-alpha in name", "$A1", map[string]string{"A1": "ok"}, "ok"},
		{"non-alpha after", "$A$B", map[string]string{"A": "a", "B": "b"}, "ab"},
		{"with underscore", "$_VAR", map[string]string{"_VAR": "ok"}, "ok"},
		{"with digit only", "$1", map[string]string{"1": "bad"}, "$1"},
		{"escaped dollar", "$$VAR", map[string]string{"VAR": "x"}, "$VAR"},
		{"escaped brace", "$${VAR}", map[string]string{"VAR": "x"}, "${VAR}"},
		{"empty var", "$EMPTY", map[string]string{"EMPTY": ""}, ""},
		{"unicode", "привет $USER", map[string]string{"USER": "мир"}, "привет мир"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandString(tt.s, tt.vars)
			if got != tt.want {
				t.Errorf("ExpandString(%q, %v) = %q, want %q", tt.s, tt.vars, got, tt.want)
			}
		})
	}
}

func TestExpandSlice(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		vars  map[string]string
		want  []string
	}{
		{"empty", []string{}, nil, []string{}},
		{"nil", nil, map[string]string{"A": "1"}, nil},
		{"single", []string{"$A"}, map[string]string{"A": "1"}, []string{"1"}},
		{"multiple", []string{"$A", "$B"}, map[string]string{"A": "a", "B": "b"}, []string{"a", "b"}},
		{"mixed", []string{"$A", "raw", "${B}"}, map[string]string{"A": "1", "B": "2"}, []string{"1", "raw", "2"}},
		{"missing", []string{"$X"}, map[string]string{"Y": "y"}, []string{"$X"}},
		{"recursive", []string{"$A"}, map[string]string{"A": "$B", "B": "C"}, []string{"C"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make([]string, len(tt.slice))
			copy(got, tt.slice)
			ExpandSlice(got, tt.vars)
			if len(got) != len(tt.want) {
				t.Errorf("length mismatch: got %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("at index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExpandMap(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]string
		vars map[string]string
		want map[string]string
	}{
		{"empty", map[string]string{}, nil, map[string]string{}},
		{"nil", nil, map[string]string{"A": "1"}, nil},
		{"single", map[string]string{"key": "$A"}, map[string]string{"A": "1"}, map[string]string{"key": "1"}},
		{"multiple", map[string]string{"k1": "$A", "k2": "$B"}, map[string]string{"A": "a", "B": "b"}, map[string]string{"k1": "a", "k2": "b"}},
		{"mixed", map[string]string{"a": "$X", "b": "raw", "c": "${Y}"}, map[string]string{"X": "1", "Y": "2"}, map[string]string{"a": "1", "b": "raw", "c": "2"}},
		{"missing", map[string]string{"x": "$Z"}, map[string]string{"W": "w"}, map[string]string{"x": "$Z"}},
		{"recursive", map[string]string{"x": "$A"}, map[string]string{"A": "$B", "B": "C"}, map[string]string{"x": "C"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make(map[string]string, len(tt.m))
			for k, v := range tt.m {
				got[k] = v
			}
			ExpandMap(got, tt.vars)
			if len(got) != len(tt.want) {
				t.Errorf("length mismatch: got %v, want %v", got, tt.want)
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("for key %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestExpandStringNoAlloc(t *testing.T) {
	s := "$A $B"
	vars := map[string]string{"A": "1", "B": "2"}
	allocs := testing.AllocsPerRun(100, func() {
		ExpandString(s, vars)
	})
	if allocs > 2 {
		t.Errorf("expected at most 2 allocations, got %f", allocs)
	}
}

func TestIsAlpha(t *testing.T) {
	tests := []struct {
		c    byte
		want bool
	}{
		{'a', true},
		{'Z', true},
		{'_', true},
		{'0', false},
		{'$', false},
		{'{', false},
		{'\n', false},
	}
	for _, tt := range tests {
		if got := isAlpha(tt.c); got != tt.want {
			t.Errorf("isAlpha(%q) = %v, want %v", tt.c, got, tt.want)
		}
	}
}

func TestIsAlphaNum(t *testing.T) {
	tests := []struct {
		c    byte
		want bool
	}{
		{'a', true},
		{'Z', true},
		{'_', true},
		{'0', true},
		{'9', true},
		{'$', false},
		{'{', false},
		{'\n', false},
	}
	for _, tt := range tests {
		if got := isAlphaNum(tt.c); got != tt.want {
			t.Errorf("isAlphaNum(%q) = %v, want %v", tt.c, got, tt.want)
		}
	}
}
