//go:build windows
// +build windows

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
	"syscall"
	"unsafe"
)

type Module struct {
	h syscall.Handle
}

func Load(path string) (*Module, error) {
	h, err := syscall.LoadLibrary(path)
	if err != nil {
		return nil, errors.New("failed to load DLL " + path + ": " + err.Error())
	}
	return &Module{h: h}, nil
}

func (m *Module) Lookup(sym string) (unsafe.Pointer, error) {
	if m == nil || m.h == 0 {
		return nil, errors.New("not loaded")
	}
	proc, err := syscall.GetProcAddress(m.h, sym)
	if err != nil {
		return nil, errors.New("symbol " + sym + " not found: " + err.Error())
	}
	return unsafe.Pointer(proc), nil
}

func (m *Module) CallInitWithGoContext(goCtx GoContext) error {
	return errors.New("cplugin exectuin is not supported on Windows")
}

func (m *Module) Close() error {
	if m == nil || m.h == 0 {
		return nil
	}
	err := syscall.FreeLibrary(m.h)
	m.h = 0
	if err != nil {
		return errors.New("FreeLibrary failed")
	}
	return nil
}

func (m *Module) CallEntryBySymbol(sym string, ctx unsafe.Pointer) error {
	p, err := m.Lookup(sym)
	if err != nil {
		return err
	}
	_, _, _ = syscall.SyscallN(uintptr(p), uintptr(ctx))
	return nil
}

func (m *Module) CallModuleEntry(moduleSym string, ctx unsafe.Pointer) error {
	if m == nil || m.h == 0 {
		return errors.New("not loaded")
	}
	modStructPtr, err := m.Lookup(moduleSym)
	if err != nil {
		return err
	}

	entryFuncPtr := *(*uintptr)(modStructPtr)
	if entryFuncPtr == 0 {
		return errors.New("module entry function is nil")
	}

	_, _, _ = syscall.SyscallN(entryFuncPtr, uintptr(ctx))
	return nil
}
