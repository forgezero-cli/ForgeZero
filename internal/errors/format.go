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
	"strconv"
	"sync"
)

var bufPool = sync.Pool{New: func() any {
	b := make([]byte, 0, 256)
	return &b
}}

func GetBuf() []byte {
	return (*bufPool.Get().(*[]byte))[:0]
}

func PutBuf(b []byte) {
	if cap(b) > 4096 {
		return
	}
	bufPool.Put(&b)
}

func AppendCode(dst []byte, code Code) []byte {
	dst = append(dst, '[')
	dst = strconv.AppendUint(dst, uint64(code), 10)
	dst = append(dst, ']')
	dst = append(dst, ' ')
	dst = append(dst, CodeName(code)...)
	return dst
}

func FormatError(code Code, component, file string, line int, detail string) string {
	buf := GetBuf()
	buf = AppendCode(buf, code)
	if component != "" {
		buf = append(buf, ' ')
		buf = append(buf, component...)
	}
	if file != "" {
		buf = append(buf, ' ')
		buf = append(buf, file...)
		if line > 0 {
			buf = append(buf, ':')
			buf = strconv.AppendInt(buf, int64(line), 10)
		}
	}
	if detail != "" {
		buf = append(buf, ": "...)
		buf = append(buf, detail...)
	}
	out := string(buf)
	PutBuf(buf)
	return out
}

func FormatJSONField(dst []byte, key, value string) []byte {
	dst = append(dst, '"')
	dst = append(dst, key...)
	dst = append(dst, '"', ':', '"')
	dst = append(dst, value...)
	dst = append(dst, '"')
	return dst
}

func FormatJSONIntField(dst []byte, key string, value int64) []byte {
	dst = append(dst, '"')
	dst = append(dst, key...)
	dst = append(dst, '"', ':')
	dst = strconv.AppendInt(dst, value, 10)
	return dst
}
