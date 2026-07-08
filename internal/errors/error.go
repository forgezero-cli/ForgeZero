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

func New(code Code) *Error {
	return &Error{Code: code}
}

func NewMsg(code Code, msg string) *Error {
	return &Error{Code: code, Msg: msg}
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
