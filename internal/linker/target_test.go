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

package linker

import "testing"

func TestTargetProfileForKnownBareMetal(t *testing.T) {
	profile, ok := TargetProfileFor("baremetal")
	if !ok {
		t.Fatal("expected baremetal profile")
	}
	if profile.Name != "baremetal" {
		t.Fatalf("expected profile.Name baremetal, got %q", profile.Name)
	}
}

func TestIsBareMetalTarget(t *testing.T) {
	old := Target
	defer func() { Target = old }()

	SetTarget("baremetal")
	if !IsBareMetalTarget() {
		t.Fatal("baremetal target should be recognized as bare metal")
	}

	SetTarget("x86_64-linux-gnu")
	if IsBareMetalTarget() {
		t.Fatal("linux target should not be recognized as bare metal")
	}
}
