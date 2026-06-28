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

func TestParseUserProfile_DefaultBalanced(t *testing.T) {
	p := ParseUserProfile("")
	if p.Name != "balanced" {
		t.Fatalf("expected balanced, got %q", p.Name)
	}
}

func TestParseUserProfile_NormalizesCaseAndAliases(t *testing.T) {
	p := ParseUserProfile("PoWeReD")
	if p.Name != "performance" {
		t.Fatalf("expected performance, got %q", p.Name)
	}

	p = ParseUserProfile("eco")
	if p.Name != "power-saver" {
		t.Fatalf("expected power-saver, got %q", p.Name)
	}
}

func TestOptimizationFlag(t *testing.T) {
	if ParseUserProfile("balanced").OptimizationFlag() != "-O2" {
		t.Fatalf("balanced optimization flag mismatch")
	}
	if ParseUserProfile("performance").OptimizationFlag() != "-O3" {
		t.Fatalf("performance optimization flag mismatch")
	}
	if ParseUserProfile("power-saver").OptimizationFlag() != "-Os" {
		t.Fatalf("power-saver optimization flag mismatch")
	}
}

func TestEffectiveJobs_UsesRequestedJobsWhenPositive(t *testing.T) {
	jobs := ParseUserProfile("performance").EffectiveJobs(8)
	if jobs != 8 {
		t.Fatalf("expected requested jobs to be preserved, got %d", jobs)
	}
}

func TestDefaultJobs_IsAtLeastOne(t *testing.T) {
	p := ParseUserProfile("power-saver")
	if p.DefaultJobs() != 1 {
		t.Fatalf("expected power-saver DefaultJobs=1, got %d", p.DefaultJobs())
	}
}
