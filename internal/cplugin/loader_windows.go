//go:build windows
// +build windows

package cplugin

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

type Module struct {
	h syscall.Handle
}

func Load(path string) (*Module, error) {
	h, err := syscall.LoadLibrary(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load DLL %s: %w", path, err)
	}
	return &Module{h: h}, nil
}

func (m *Module) Lookup(sym string) (unsafe.Pointer, error) {
	if m == nil || m.h == 0 {
		return nil, errors.New("not loaded")
	}
	proc, err := syscall.GetProcAddress(m.h, sym)
	if err != nil {
		return nil, fmt.Errorf("symbol %s not found: %w", sym, err)
	}
	return unsafe.Pointer(proc), nil
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
