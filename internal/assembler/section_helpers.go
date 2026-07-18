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

func (s *sectionState) AppendByte(b byte) {
	n := len(s.data)
	if n+1 <= cap(s.data) {
		s.data = s.data[:n+1]
		s.data[n] = b
		return
	}
	s.data = append(s.data, b)
}

func (s *sectionState) AppendBytes(p []byte) {
	if len(p) == 0 {
		return
	}
	n := len(s.data)
	need := n + len(p)
	if need <= cap(s.data) {
		s.data = s.data[:need]
		copy(s.data[n:], p)
		return
	}
	s.data = append(s.data, p...)
}

func (s *sectionState) AppendZeros(count int) {
	if count <= 0 {
		return
	}
	n := len(s.data)
	need := n + count
	if need <= cap(s.data) {
		s.data = s.data[:need]
		for i := n; i < need; i++ {
			s.data[i] = 0
		}
		return
	}
	for i := 0; i < count; i++ {
		s.data = append(s.data, 0)
	}
}
