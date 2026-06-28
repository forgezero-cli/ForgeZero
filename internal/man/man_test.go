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

package man

import (
	"strings"
	"testing"
)

func TestGenerateManPage(t *testing.T) {
	version := "1.7.1"
	page := GenerateManPage(version)
	if !strings.Contains(page, ".TH fz") {
		t.Error("missing .TH header")
	}
	if !strings.Contains(page, version) {
		t.Errorf("missing version %s", version)
	}
	if !strings.Contains(page, ".SH NAME") {
		t.Error("missing NAME section")
	}
	if !strings.Contains(page, ".SH SYNOPSIS") {
		t.Error("missing SYNOPSIS section")
	}
	if !strings.Contains(page, ".SH OPTIONS") {
		t.Error("missing OPTIONS section")
	}
}
