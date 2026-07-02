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

package hybrid

import "testing"

func TestArgs(t *testing.T) {
	cases := []struct {
		tool string
		want string
	}{
		{"/usr/bin/xorriso", "-isohybrid-gpt-basdat"},
		{"xorriso", "-isohybrid-gpt-basdat"},
		{"/usr/bin/genisoimage", "-isohybrid"},
		{"mkisofs", "-isohybrid"},
	}
	for _, c := range cases {
		got := Args(c.tool)
		if len(got) != 1 || got[0] != c.want {
			t.Errorf("Args(%q) = %v, want [%s]", c.tool, got, c.want)
		}
	}
}

func TestArgsUnknown(t *testing.T) {
	if got := Args("/usr/bin/unknowntool"); got != nil {
		t.Errorf("expected nil for unknown tool, got %v", got)
	}
}
