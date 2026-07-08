//go:build !unix
// +build !unix

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

package utils

import (
	"io"
	"os"

	fzvfs "github.com/forgezero-cli/ForgeZero/internal/fs"
)

func hashRawFileDigest(path string) ([32]byte, error) {
	var out [32]byte
	if fileSystem() != fzvfs.Default {
		f, err := openVerified(path)
		if err != nil {
			return out, ErrHashOpen
		}
		hasher := getKeyedHasher()
		var buf [65536]byte
		if _, err := io.CopyBuffer(hasher, f, buf[:]); err != nil {
			putKeyedHasher(hasher)
			f.Close()
			return out, err
		}
		if cerr := f.Close(); cerr != nil {
			putKeyedHasher(hasher)
			return out, cerr
		}
		digest := hasher.Digest()
		if _, err := digest.Read(out[:]); err != nil {
			putKeyedHasher(hasher)
			return out, err
		}
		putKeyedHasher(hasher)
		return out, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return out, ErrHashOpen
	}
	defer f.Close()

	hasher := getKeyedHasher()
	var buf [65536]byte
	if _, err := io.CopyBuffer(hasher, f, buf[:]); err != nil {
		putKeyedHasher(hasher)
		return out, err
	}
	digest := hasher.Digest()
	if _, err := digest.Read(out[:]); err != nil {
		putKeyedHasher(hasher)
		return out, err
	}
	putKeyedHasher(hasher)
	return out, nil
}
