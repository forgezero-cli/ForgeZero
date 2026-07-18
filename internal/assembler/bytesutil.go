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

package assembler

import (
	"bytes"
	"errors"
)

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalWord(token []byte, literal string) bool {
	if len(token) != len(literal) {
		return false
	}
	for i := 0; i < len(token); i++ {
		if token[i] != literal[i] {
			return false
		}
	}
	return true
}

func readToken(line []byte) ([]byte, []byte) {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	start := i
	for i < len(line) && line[i] != ' ' && line[i] != '\t' {
		i++
	}
	return line[start:i], line[i:]
}

func (p *parser) splitArgs(data []byte) [][]byte {
	p.argsPool = p.argsPool[:0]

	start := 0
	in := false
	var q byte
	for i := 0; i < len(data); i++ {
		c := data[i]
		if in {
			if c == q {
				in = false
			}
			continue
		}
		if c == '"' || c == '\'' {
			in = true
			q = c
			continue
		}
		if c == ',' {
			p.argsPool = append(p.argsPool, data[start:i])
			start = i + 1
		}
	}
	if start <= len(data) {
		p.argsPool = append(p.argsPool, data[start:])
	}
	return p.argsPool
}

var asciiSpaceTable = [256]bool{
	' ':  true,
	'\t': true,
	'\r': true,
	'\n': true,
}

func trimSpace(data []byte) []byte {
	start := 0
	end := len(data)

	for start < end && asciiSpaceTable[data[start]] {
		start++
	}

	for end > start && asciiSpaceTable[data[end-1]] {
		end--
	}

	return data[start:end]
}

func removeComment(line []byte) []byte {
	idx := bytes.IndexAny(line, ";#")

	if idx == -1 {
		return line
	}

	return line[:idx]
}

func parseString(token []byte) ([]byte, error) {
	if len(token) < 2 {
		return nil, errors.New("invalid string")
	}
	quote := token[0]
	if token[len(token)-1] != quote {
		return nil, errors.New("unterminated string")
	}
	return token[1 : len(token)-1], nil
}

func parseNumber(token []byte) (uint64, error) {
	if len(token) == 0 {
		return 0, errors.New("empty number")
	}
	base := 10
	i := 0
	if token[0] == '-' {
		i = 1
	}
	if i+1 < len(token) && token[i] == '0' {
		switch token[i+1] {
		case 'x', 'X':
			base = 16
			i += 2
		case 'b', 'B':
			base = 2
			i += 2
		case 'o', 'O':
			base = 8
			i += 2
		}
	}
	var v uint64
	for ; i < len(token); i++ {
		c := token[i]
		var digit uint64
		if c >= '0' && c <= '9' {
			digit = uint64(c - '0')
		} else if base == 16 && c >= 'a' && c <= 'f' {
			digit = uint64(10 + c - 'a')
		} else if base == 16 && c >= 'A' && c <= 'F' {
			digit = uint64(10 + c - 'A')
		} else {
			return 0, errors.New("invalid digit")
		}
		if digit >= uint64(base) {
			return 0, errors.New("invalid digit for base")
		}
		v = v*uint64(base) + digit
	}
	return v, nil
}

func alignValue(value, align uint64) uint64 {
	if align == 0 {
		return value
	}
	mask := align - 1
	return (value + mask) &^ mask
}

func alignOut(out []byte, align uint64) []byte {
	if align == 0 {
		return out
	}
	pad := int((align - uint64(len(out))%align) % align)
	if pad == 0 {
		return out
	}
	n := len(out)
	need := n + pad
	if need <= cap(out) {
		out = out[:need]
		for i := n; i < need; i++ {
			out[i] = 0
		}
		return out
	}
	nb := make([]byte, need)
	copy(nb, out)
	return nb
}

func alignOutOffset(offset int, align uint64) int {
	if align == 0 {
		return offset
	}
	mask := int(align - 1)
	return (offset + mask) &^ mask
}

func appendByte(out []byte, v byte) []byte {
	n := len(out)
	if n+1 <= cap(out) {
		out = out[:n+1]
		out[n] = v
		return out
	}
	return append(out, v)
}

func containsFold(s, substr string) bool {
	slen := len(s)
	subLen := len(substr)
	if subLen == 0 || subLen > slen {
		return false
	}
	for i := 0; i <= slen-subLen; i++ {
		match := true
		for j := 0; j < subLen; j++ {
			sc := s[i+j]
			pc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 'a' - 'A'
			}
			if pc >= 'A' && pc <= 'Z' {
				pc += 'a' - 'A'
			}
			if sc != pc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func appendUint16(out []byte, v uint16) []byte {
	n := len(out)
	if n+2 <= cap(out) {
		out = out[:n+2]
		out[n] = byte(v)
		out[n+1] = byte(v >> 8)
		return out
	}
	return append(out, byte(v), byte(v>>8))
}

func appendUint32(out []byte, v uint32) []byte {
	n := len(out)
	if n+4 <= cap(out) {
		out = out[:n+4]
		out[n] = byte(v)
		out[n+1] = byte(v >> 8)
		out[n+2] = byte(v >> 16)
		out[n+3] = byte(v >> 24)
		return out
	}
	return append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func appendUint64(out []byte, v uint64) []byte {
	n := len(out)
	if n+8 <= cap(out) {
		out = out[:n+8]
		out[n] = byte(v)
		out[n+1] = byte(v >> 8)
		out[n+2] = byte(v >> 16)
		out[n+3] = byte(v >> 24)
		out[n+4] = byte(v >> 32)
		out[n+5] = byte(v >> 40)
		out[n+6] = byte(v >> 48)
		out[n+7] = byte(v >> 56)
		return out
	}
	return append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24), byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}
