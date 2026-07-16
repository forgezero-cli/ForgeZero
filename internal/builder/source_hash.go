/*
 * Copyright (c) 2026 ForgeZero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package builder

import (
	"io"
	"os"
	"path/filepath"

	"github.com/forgezero-cli/ForgeZero/internal/hashpool"
)

var sourceHashes = make(map[string][32]byte)

func refreshSourceHashes(dirs []string) error {
	buf := make([]byte, 32*1024)
	sourceHashes = make(map[string][32]byte)
	for _, root := range dirs {
		if root == "" {
			continue
		}
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				fi, serr := os.Stat(path)
				if serr != nil {
					return serr
				}
				if fi.IsDir() {
					return nil
				}
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			h := hashpool.GetHasher()
			defer hashpool.PutHasher(h)
			for {
				n, rerr := f.Read(buf)
				if n > 0 {
					_, _ = h.Write(buf[:n])
				}
				if rerr == io.EOF {
					break
				}
				if rerr != nil {
					_ = f.Close()
					return rerr
				}
			}
			_ = f.Close()
			var sum [32]byte
			h.Sum(sum[:0])
			sourceHashes[path] = sum
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
