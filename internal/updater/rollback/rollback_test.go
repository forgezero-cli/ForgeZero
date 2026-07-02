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

package rollback

import (
	"errors"
	"testing"
)

func swap(r func() error, i func(string) error) func() {
	origR, origI := restore, install
	if r != nil {
		restore = r
	}
	if i != nil {
		install = i
	}
	return func() {
		restore = origR
		install = origI
	}
}

func TestToEmptyVersion(t *testing.T) {
	if err := To(""); err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestToDelegates(t *testing.T) {
	got := ""
	restoreFn := func() error { return nil }
	installFn := func(v string) error { got = v; return nil }
	defer swap(restoreFn, installFn)()
	if err := To("1.2.3"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.2.3" {
		t.Errorf("expected version passed through, got %q", got)
	}
}

func TestToPropagatesError(t *testing.T) {
	installFn := func(string) error { return errors.New("boom") }
	defer swap(func() error { return nil }, installFn)()
	if err := To("9.9.9"); err == nil {
		t.Fatal("expected install error")
	}
}

func TestRunDelegates(t *testing.T) {
	called := false
	restoreFn := func() error { called = true; return nil }
	defer swap(restoreFn, nil)()
	if err := Run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected restore to be called")
	}
}

func TestRunPropagatesError(t *testing.T) {
	restoreFn := func() error { return errors.New("no backup") }
	defer swap(restoreFn, nil)()
	if err := Run(); err == nil {
		t.Fatal("expected restore error")
	}
}
