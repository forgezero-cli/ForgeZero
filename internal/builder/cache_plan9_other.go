//go:build !amd64 && !arm64

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

func pathBuffer_appendStringPlan9(p *pathBuffer, s string) {
	p.appendString(s)
}

func pathBuffer_appendBytePlan9(p *pathBuffer, b byte) {
	p.appendByte(b)
}

func pathBuffer_appendBytesPlan9(p *pathBuffer, b []byte) {
	p.appendBytes(b)
}

func joinPathPlan9(base, name string) string {
	return joinPathPlan9Fallback(base, name)
}

func buildCacheKeyPlan9(hash string, debug bool, mode string) string {
	return buildCacheKeyPlan9Fallback(hash, debug, mode)
}

func cacheEntryPathPlan9(dir, key string) string {
	return cacheEntryPathPlan9Fallback(dir, key)
}
