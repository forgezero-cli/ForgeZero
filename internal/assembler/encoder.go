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

import "sync"

type Encoder struct{
    buf []byte
}

var encPool = sync.Pool{
    New: func() any {
        b := make([]byte, 0, 256)
        return &Encoder{buf: b}
    },
}

func GetEncoder() *Encoder {
    e := encPool.Get().(*Encoder)
    e.buf = e.buf[:0]
    return e
}

func PutEncoder(e *Encoder) {
    encPool.Put(e)
}

func (e *Encoder) WriteByte(b byte) {
    e.buf = append(e.buf, b)
}

func (e *Encoder) Write(p []byte) {
    e.buf = append(e.buf, p...)
}

func (e *Encoder) Bytes() []byte { return e.buf }
