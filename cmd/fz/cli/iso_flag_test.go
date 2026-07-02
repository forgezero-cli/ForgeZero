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

package cli

import "testing"

func TestISOFlagBareEnables(t *testing.T) {
	var f ISOFlag
	if err := f.Set("true"); err != nil {
		t.Fatal(err)
	}
	if !f.Enabled {
		t.Error("expected enabled")
	}
	if f.Dir != "" {
		t.Errorf("expected empty dir, got %q", f.Dir)
	}
}

func TestISOFlagWithDir(t *testing.T) {
	var f ISOFlag
	if err := f.Set("./isoroot"); err != nil {
		t.Fatal(err)
	}
	if !f.Enabled {
		t.Error("expected enabled")
	}
	if f.Dir != "./isoroot" {
		t.Errorf("expected ./isoroot, got %q", f.Dir)
	}
	if f.String() != "./isoroot" {
		t.Errorf("String mismatch: %q", f.String())
	}
}

func TestISOFlagIsBool(t *testing.T) {
	var f ISOFlag
	if !f.IsBoolFlag() {
		t.Error("expected IsBoolFlag true")
	}
}

func TestISOFlagNilString(t *testing.T) {
	var f *ISOFlag
	if f.String() != "" {
		t.Error("expected empty string for nil receiver")
	}
}
