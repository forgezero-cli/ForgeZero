//go:build !cgo && !windows
// +build !cgo,!windows

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
