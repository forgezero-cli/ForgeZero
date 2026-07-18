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

package stdio

import (
	"errors"
	"testing"
)

func TestAppendIntUintHex(t *testing.T) {
	b := AppendInt(nil, -42)
	if string(b) == "" {
		t.Fatal("AppendInt returned empty")
	}
	b = AppendUint(nil, 0)
	if string(b) != "0" {
		t.Fatal("AppendUint for 0 failed")
	}
	b = AppendHex(nil, 255, false)
	if string(b) == "" {
		t.Fatal("AppendHex failed")
	}
}

func TestAppendAnyFormatAppend(t *testing.T) {
	b := AppendAny(nil, "s")
	if string(b) != "s" {
		t.Fatal("AppendAny string failed")
	}
	b = AppendAny(nil, errors.New("e"))
	if string(b) == "" {
		t.Fatal("AppendAny error failed")
	}
	out := FormatAppend(nil, "x=%d s=%s", 7, "x")
	if len(out) == 0 {
		t.Fatal("FormatAppend returned empty")
	}
}

func TestErrorf(t *testing.T) {
	err := Errorf("err %d", 5)
	if err == nil || err.Error() == "" {
		t.Fatal("Errorf failed")
	}
}
