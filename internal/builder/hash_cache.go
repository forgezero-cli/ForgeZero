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
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
)

func loadHashCache(cacheDir string) (map[string][32]byte, error) {
	if cacheDir == "" {
		return nil, nil
	}
	path := filepath.Join(cacheDir, ".fz-build-hash-cache")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	m := make(map[string][32]byte)
	var lenBuf [2]byte
	var hashBuf [32]byte
	for {
		_, err := io.ReadFull(f, lenBuf[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		pathLen := binary.LittleEndian.Uint16(lenBuf[:])
		pathBytes := make([]byte, pathLen)
		if _, err := io.ReadFull(f, pathBytes); err != nil {
			return nil, err
		}
		if _, err := io.ReadFull(f, hashBuf[:]); err != nil {
			return nil, err
		}
		m[string(pathBytes)] = hashBuf
	}
	return m, nil
}

func saveHashCache(cacheDir string, m map[string][32]byte) error {
	if cacheDir == "" || m == nil {
		return nil
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(cacheDir, ".fz-build-hash-cache")
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(tmp)
	}()
	var lenBuf [2]byte
	var hashBuf [32]byte
	for k, v := range m {
		if len(k) > 65535 {
			continue
		}
		binary.LittleEndian.PutUint16(lenBuf[:], uint16(len(k)))
		if _, err := f.Write(lenBuf[:]); err != nil {
			return err
		}
		if _, err := f.Write([]byte(k)); err != nil {
			return err
		}
		copy(hashBuf[:], v[:])
		if _, err := f.Write(hashBuf[:]); err != nil {
			return err
		}
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}