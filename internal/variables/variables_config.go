// SPDX-LICENSE-INDITIFIER MIT
// AUTHOR: ALEXVOSTE

package variables

import "strings"

func ExpandString(s string, vars map[string]string) string {
	if len(vars) == 0 || s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) * 2)
	for i := 0; i < len(s); {
		if s[i] == '$' && i+1 < len(s) {
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
