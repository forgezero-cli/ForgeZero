//go:build !cgo && !windows
// +build !cgo,!windows

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

package cplugin

import (
	"errors"
	"unsafe"
)

type Module struct {
	h unsafe.Pointer
}

func Load(path string) (*Module, error) {
	return nil, errors.New("cplugin is disabled in non-CGO builds")
}

func (m *Module) Lookup(sym string) (unsafe.Pointer, error) {
	return nil, errors.New("not supported")
}

func (m *Module) Close() error {
	return nil
}

func (m *Module) CallEntryBySymbol(sym string, ctx unsafe.Pointer) error {
	return errors.New("not supported")
}

func (m *Module) CallModuleEntry(moduleSym string, ctx unsafe.Pointer) error {
	return errors.New("not supported")
}

func (m *Module) CallInitWithGoContext(goCtx GoContext) error {
	return errors.New("cplugin execution is not supported without CGO")
}
