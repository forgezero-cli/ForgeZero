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
	"bytes"
	"os"
	"runtime"
	"strconv"
)

type Level int

const (
	LevelNote Level = iota
	LevelWarning
	LevelError
	LevelPanic
)

type Diagnostic struct {
	Level    Level
	FilePath string
	Line     int
	Param    string
	Message  string
	FixHint  string
	Source   []byte
}

func getBuf() []byte {
	buf := bufPool.Get().(*[]byte)
	return (*buf)[:0]
}

func putBuf(buf []byte) {
	if cap(buf) > 4096 {
		return
	}
	b := &buf
	bufPool.Put(b)
}

func (d Diagnostic) Log() {
	buf := getBuf()
	buf = d.appendFormatted(buf)
	_, _ = os.Stderr.Write(buf)
	putBuf(buf)
	if d.Level == LevelError || d.Level == LevelPanic {
		if d.Level == LevelPanic {
			buf := make([]byte, 0, 256)
			buf = append(buf, "\n[panic] internal compiler failure\n"...)
			_, _ = os.Stderr.Write(buf)
			_, _ = os.Stderr.Write([]byte("\n"))
			runtime.Breakpoint()
		}
		os.Exit(1)
	}
}

func (d Diagnostic) appendFormatted(dst []byte) []byte {
	prefix, color := levelPrefix(d.Level)
	dst = append(dst, color...)
	dst = append(dst, prefix...)
	dst = append(dst, "\x1b[0m"...)
	if d.FilePath != "" {
		dst = append(dst, " in "...)
		dst = append(dst, d.FilePath...)
		if d.Line > 0 {
			dst = append(dst, " (line "...)
			dst = strconv.AppendInt(dst, int64(d.Line), 10)
			dst = append(dst, ')')
		}
		dst = append(dst, '\n')
	}
	if d.FilePath != "" && d.Line > 0 {
		if data, err := os.ReadFile(d.FilePath); err == nil {
			dst = appendLineError(dst, data, d.Line, d.Param, d.Message, d.FixHint, d.Level)
			return dst
		}
	}
	if d.Param != "" {
		dst = append(dst, "  parameter: "...)
		dst = append(dst, d.Param...)
		dst = append(dst, '\n')
	}
	if d.Message != "" {
		dst = append(dst, "  detail: "...)
		dst = append(dst, d.Message...)
		dst = append(dst, '\n')
	}
	if d.FixHint != "" {
		dst = append(dst, "  fix: "...)
		dst = append(dst, d.FixHint...)
		dst = append(dst, '\n')
	}
	if len(d.Source) > 0 {
		dst = append(dst, "  source: "...)
		dst = append(dst, d.Source...)
		dst = append(dst, '\n')
	}
	return dst
}

func levelPrefix(level Level) ([]byte, []byte) {
	switch level {
	case LevelNote:
		return []byte("NOTE"), []byte("\x1b[36m")
	case LevelWarning:
		return []byte("WARNING"), []byte("\x1b[33m")
	case LevelError:
		return []byte("ERROR"), []byte("\x1b[31m")
	case LevelPanic:
		return []byte("PANIC"), []byte("\x1b[1;31m")
	default:
		return []byte("INFO"), []byte("\x1b[37m")
	}
}

func RenderLineError(file []byte, lineNum int, param, msg, hint string) []byte {
	buf := getBuf()
	buf = appendLineError(buf, file, lineNum, param, msg, hint, LevelError)
	out := append([]byte(nil), buf...)
	putBuf(buf)
	return out
}

func appendLineError(dst []byte, file []byte, lineNum int, param, msg, hint string, level Level) []byte {
	if len(file) == 0 {
		return append(dst, "<empty source>"...)
	}
	line := findLine(file, lineNum)
	if line == nil {
		return append(dst, "<line not found>"...)
	}
	prefix, color := levelPrefix(level)
	dst = append(dst, color...)
	dst = append(dst, prefix...)
	dst = append(dst, "\x1b[0m\n"...)
	dst = append(dst, "  "...)
	dst = strconv.AppendInt(dst, int64(lineNum), 10)
	dst = append(dst, " | "...)
	dst = append(dst, line...)
	dst = append(dst, '\n')
	dst = append(dst, "    | "...)
	col := paramColumn(line, param)
	if col >= 0 {
		for i := 0; i < col; i++ {
			dst = append(dst, ' ')
		}
		width := len(param)
		if width <= 0 {
			width = 1
		}
		for i := 0; i < width; i++ {
			dst = append(dst, '^')
		}
	} else {
		dst = append(dst, '^')
	}
	dst = append(dst, '\n')
	if param != "" {
		dst = append(dst, "  parameter: "...)
		dst = append(dst, param...)
		dst = append(dst, '\n')
	}
	if msg != "" {
		dst = append(dst, "  detail: "...)
		dst = append(dst, msg...)
		dst = append(dst, '\n')
	}
	if hint != "" {
		dst = append(dst, "  fix: "...)
		dst = append(dst, hint...)
		dst = append(dst, '\n')
	}
	return dst
}

func paramColumn(line []byte, param string) int {
	if len(line) == 0 || param == "" {
		return -1
	}
	paramBytes := []byte(param)
	if len(paramBytes) > len(line) {
		return -1
	}
	for i := 0; i <= len(line)-len(paramBytes); i++ {
		if !bytes.EqualFold(line[i:i+len(paramBytes)], paramBytes) {
			continue
		}
		return i
	}
	return -1
}

func findLine(file []byte, lineNum int) []byte {
	if lineNum <= 0 {
		return nil
	}
	line := 1
	start := 0
	for i := 0; i < len(file); i++ {
		if file[i] == '\n' {
			if line == lineNum {
				return trimLine(file[start:i])
			}
			line++
			start = i + 1
		}
	}
	if line == lineNum {
		return trimLine(file[start:])
	}
	return nil
}

func trimLine(line []byte) []byte {
	return bytesTrimSpace(line)
}

func bytesTrimSpace(b []byte) []byte {
	start := 0
	end := len(b)
	for start < end && (b[start] == ' ' || b[start] == '\t' || b[start] == '\r' || b[start] == '\n') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t' || b[end-1] == '\r' || b[end-1] == '\n') {
		end--
	}
	return b[start:end]
}

func AppendMsg(buf []byte, code Code, parts ...string) []byte {
	buf = AppendCode(buf, code)
	for _, p := range parts {
		if p == "" {
			continue
		}
		buf = append(buf, ':', ' ')
		buf = append(buf, p...)
	}
	return buf
}

func FromBuf(code Code, buf []byte) *Error {
	return &Error{Code: code, Msg: string(buf)}
}

func GetCode(err error) Code {
	if err == nil {
		return CodeOK
	}
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return CodeGeneric
}

func Is(err error, code Code) bool {
	return GetCode(err) == code
}

func New(code Code) *Error {
	return &Error{Code: code}
}

func NewMsg(code Code, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

type Error struct {
	Code Code
	Msg  string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg != "" {
		return e.Msg
	}
	return CodeName(e.Code)
}
