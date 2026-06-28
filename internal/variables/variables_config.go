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

import "strings"

const escMarker = "\x01"

func ExpandString(s string, vars map[string]string) string {
	if len(vars) == 0 || s == "" {
		return s
	}
	prev := s
	for i := 0; i < 10; i++ {
		cur := expandOne(prev, vars)
		if cur == prev {
			return strings.ReplaceAll(cur, escMarker, "$")
		}
		prev = cur
	}
	return strings.ReplaceAll(prev, escMarker, "$")
}

func expandOne(s string, vars map[string]string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)
	for i := 0; i < len(s); {
		if s[i] == '$' && i+1 < len(s) {
			if s[i+1] == '$' {
				b.WriteString(escMarker)
				i += 2
				continue
			}
			if s[i+1] == '{' {
				j := i + 2
				for j < len(s) && s[j] != '}' {
					j++
				}
				if j < len(s) {
					name := s[i+2 : j]
					if val, ok := vars[name]; ok {
						b.WriteString(val)
						i = j + 1
						continue
					}
				}
				b.WriteByte(s[i])
				i++
				continue
			}
			if isAlpha(s[i+1]) {
				j := i + 1
				for j < len(s) && isAlphaNum(s[j]) {
					j++
				}
				name := s[i+1 : j]
				if val, ok := vars[name]; ok {
					b.WriteString(val)
					i = j
					continue
				}
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func ExpandSlice(slice []string, vars map[string]string) {
	if len(vars) == 0 || len(slice) == 0 {
		return
	}
	for i, s := range slice {
		slice[i] = ExpandString(s, vars)
	}
}

func ExpandMap(m map[string]string, vars map[string]string) {
	if len(vars) == 0 || len(m) == 0 {
		return
	}
	for k, v := range m {
		m[k] = ExpandString(v, vars)
	}
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isAlphaNum(c byte) bool {
	return isAlpha(c) || (c >= '0' && c <= '9')
}
