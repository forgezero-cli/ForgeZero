/*
 *   Copyright (c) 2026 forgezero-cli
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
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSourcePooledLargeFile(t *testing.T) {
	const size = 2*1024*1024 + 123
	want := make([]byte, size)
	for index := range want {
		want[index] = byte(index*31 + 9)
	}
	path := filepath.Join(t.TempDir(), "large.s")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}
	got, release, err := loadSourcePooled(path)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if !bytes.Equal(got, want) {
		t.Fatal("large source buffer mismatch")
	}
}
