//go:build windows

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

package fs

import (
	"os"
	"time"
)

func renameAtomic(oldpath, newpath string) error {
	var last error
	for attempt := 0; attempt < 8; attempt++ {
		if err := os.Rename(oldpath, newpath); err == nil {
			return nil
		} else {
			last = err
			time.Sleep(time.Millisecond * time.Duration(10*(attempt+1)))
		}
	}
	return last
}
