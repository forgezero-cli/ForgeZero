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
	"sync"
)

type Encoder struct {
	buf  []byte
	next *Encoder
}

var encPool sync.Pool

func init() {
	encPool.New = func() any {
		b := make([]byte, 0, 4096)
		return &Encoder{buf: b}
	}
}

func GetEncoder() *Encoder {
	v := encPool.Get()
	if v == nil {
		b := make([]byte, 0, 4096)
		return &Encoder{buf: b}
	}
	e := v.(*Encoder)
	e.next = nil
	e.buf = e.buf[:0]
	return e
}

func (e *Encoder) Reserve(n int) {
	free := cap(e.buf) - len(e.buf)
	if free >= n {
		return
	}
	need := len(e.buf) + n
	newCap := cap(e.buf) * 2
	if newCap < 4096 {
		newCap = 4096
	}
	if newCap < need {
		newCap = need
	}
	nb := make([]byte, len(e.buf), newCap)
	copy(nb, e.buf)
	e.buf = nb
}

func PutEncoder(e *Encoder) {
	e.buf = e.buf[:0]
	e.next = nil
	encPool.Put(e)
}

func (e *Encoder) WriteByte(b byte) error {
	n := len(e.buf)
	if n+1 <= cap(e.buf) {
		e.buf = e.buf[:n+1]
		e.buf[n] = b
		return nil
	}
	writeByteGrow(e, b)
	return nil
}

func (e *Encoder) Write(p []byte) {
	n := len(e.buf)
	need := n + len(p)
	if need <= cap(e.buf) {
		e.buf = e.buf[:need]
		copy(e.buf[n:], p)
		return
	}
	newCap := cap(e.buf) * 2
	if newCap < 256 {
		newCap = 256
	}
	if newCap < need {
		newCap = need
	}
	nb := make([]byte, n, newCap)
	copy(nb, e.buf)
	nb = nb[:need]
	copy(nb[n:], p)
	e.buf = nb
}

func writeByteGrow(e *Encoder, b byte) {
	e.buf = append(e.buf, b)
}

func writeByteAsm(e *Encoder, b byte) {
	writeByteGrow(e, b)
}

func (e *Encoder) Bytes() []byte { return e.buf }
