//go:build !linux

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

import "os"

func emitDecoyObject(path string) error {
	const size = 4096
	data := make([]byte, size)
	copy(data[:4], []byte{0x7f, 'E', 'L', 'F'})
	for i := 4; i < size; i++ {
		data[i] = byte((i * 37) ^ 0x55)
	}
	return os.WriteFile(path, data, 0o600)
}
