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

package profiles

import "testing"

func TestParseUserProfile_DefaultBalanced_EmptyInput(t *testing.T) {
	p := ParseUserProfile("")
	if p.Name != "balanced" {
		t.Fatalf("expected balanced, got %q", p.Name)
	}
}

func TestParseUserProfile_NormalizesCase_SeparateFile(t *testing.T) {
	p := ParseUserProfile("PoWeReD")
	if p.Name != "performance" {
		t.Fatalf("expected performance, got %q", p.Name)
	}
}
