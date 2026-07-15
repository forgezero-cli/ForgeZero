//go:build arm64

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

package builder

import "os"

func pathBuffer_appendStringPlan9(p *pathBuffer, s string) {
	if len(s) == 0 {
		return
	}
	if p.extra != nil {
		p.extra = append(p.extra, s...)
		return
	}
	if len(s)+p.n <= len(p.buf) {
		copy(p.buf[p.n:], s)
		p.n += len(s)
		return
	}
	p.extra = append(p.extra, p.buf[:p.n]...)
	p.extra = append(p.extra, s...)
}

func pathBuffer_appendBytePlan9(p *pathBuffer, b byte) {
	if p.extra != nil {
		p.extra = append(p.extra, b)
		return
	}
	if p.n < len(p.buf) {
		p.buf[p.n] = b
		p.n++
		return
	}
	p.extra = append(p.extra, p.buf[:p.n]...)
	p.extra = append(p.extra, b)
}

func pathBuffer_appendBytesPlan9(p *pathBuffer, b []byte) {
	if len(b) == 0 {
		return
	}
	if p.extra != nil {
		p.extra = append(p.extra, b...)
		return
	}
	if len(b)+p.n <= len(p.buf) {
		copy(p.buf[p.n:], b)
		p.n += len(b)
		return
	}
	p.extra = append(p.extra, p.buf[:p.n]...)
	p.extra = append(p.extra, b...)
}

func joinPathPlan9(base, name string) string {
	var pb pathBuffer
	pathBuffer_appendStringPlan9(&pb, base)
	if len(base) > 0 && base[len(base)-1] != byte(os.PathSeparator) {
		pathBuffer_appendBytePlan9(&pb, byte(os.PathSeparator))
	}
	pathBuffer_appendStringPlan9(&pb, name)
	return pb.String()
}

func buildCacheKeyPlan9(hash string, debug bool, mode string) string {
	var pb pathBuffer
	pathBuffer_appendStringPlan9(&pb, hash)
	pathBuffer_appendBytePlan9(&pb, '_')
	if debug {
		pathBuffer_appendBytePlan9(&pb, '1')
	} else {
		pathBuffer_appendBytePlan9(&pb, '0')
	}
	pathBuffer_appendBytePlan9(&pb, '_')
	pathBuffer_appendStringPlan9(&pb, mode)
	return pb.String()
}

func cacheEntryPathPlan9(dir, key string) string {
	var pb pathBuffer
	pathBuffer_appendStringPlan9(&pb, dir)
	if len(dir) > 0 && dir[len(dir)-1] != byte(os.PathSeparator) {
		pathBuffer_appendBytePlan9(&pb, byte(os.PathSeparator))
	}
	pathBuffer_appendStringPlan9(&pb, key)
	return pb.String()
}
