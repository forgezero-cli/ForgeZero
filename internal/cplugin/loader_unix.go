//go:build !windows
// +build !windows

package cplugin

/*
#include <stdlib.h>
#include <dlfcn.h>
#include "fz_module.h"

static void* open_lib(const char* path) { return dlopen(path, RTLD_NOW|RTLD_LOCAL); }
static void* symbol(void* handle, const char* name) { return dlsym(handle, name); }
static int close_lib(void* handle) { return dlclose(handle); }
static const char* dlerr() { const char* e = dlerror(); return e?e:""; }
static void call_entry(void* fp, void* ctx) { ((fz_entry_t)fp)(ctx); }
static int call_module_entry_by_modsym(void* handle, const char* modsym, void* ctx) {
    void* mod = dlsym(handle, modsym);
    if (!mod) return -1;
    fz_module_info* m = (fz_module_info*)mod;
    if (!m->entry) return -2;
    m->entry(ctx);
    return 0;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

type Module struct{ h unsafe.Pointer }

func Load(path string) (*Module, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	h := C.open_lib(cpath)
	if h == nil {
		return nil, errors.New(C.GoString(C.dlerr()))
	}
	return &Module{h: h}, nil
}

func (m *Module) Lookup(sym string) (unsafe.Pointer, error) {
	if m == nil || m.h == nil {
		return nil, errors.New("not loaded")
	}
	cs := C.CString(sym)
	defer C.free(unsafe.Pointer(cs))
	p := C.symbol(m.h, cs)
	if p == nil {
		return nil, errors.New(C.GoString(C.dlerr()))
	}
	return unsafe.Pointer(p), nil
}

func (m *Module) Close() error {
	if m == nil || m.h == nil {
		return nil
	}
	r := C.close_lib(m.h)
	m.h = nil
	if r != 0 {
		return errors.New("dlclose failed")
	}
	return nil
}

func (m *Module) CallEntryBySymbol(sym string, ctx unsafe.Pointer) error {
	p, err := m.Lookup(sym)
	if err != nil {
		return err
	}
	C.call_entry(p, ctx)
	return nil
}

func (m *Module) CallModuleEntry(moduleSym string, ctx unsafe.Pointer) error {
	if m == nil || m.h == nil {
		return errors.New("not loaded")
	}
	cs := C.CString(moduleSym)
	defer C.free(unsafe.Pointer(cs))
	r := C.call_module_entry_by_modsym(m.h, cs, ctx)
	if r != 0 {
		return errors.New("module entry call failed")
	}
	return nil
}
