//go:build (freebsd || openbsd || netbsd || dragonfly || linux || darwin) && cgo
// +build freebsd openbsd netbsd dragonfly linux darwin
// +build cgo

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

func (m *Module) CallInitWithGoContext(goCtx GoContext) error {
	if m == nil || m.h == nil {
		return errors.New("not loaded")
	}

	cPluginPath := C.CString(goCtx.PluginPath)
	cConfigPath := C.CString(goCtx.ConfigPath)
	cSourcePath := C.CString(goCtx.SourcePath)
	cDirPath := C.CString(goCtx.DirPath)
	cOutBin := C.CString(goCtx.OutBin)
	cOutObj := C.CString(goCtx.OutObj)
	cBuildType := C.CString(goCtx.BuildType)
	cTarget := C.CString(goCtx.Target)
	cToolchain := C.CString(goCtx.Toolchain)
	cMode := C.CString(goCtx.Mode)
	cCcFlags := C.CString(goCtx.CcFlags)
	cLdFlags := C.CString(goCtx.LdFlags)
	cFormat := C.CString(goCtx.Format)
	cIsolation := C.CString(goCtx.Isolation)

	defer func() {
		C.free(unsafe.Pointer(cPluginPath))
		C.free(unsafe.Pointer(cConfigPath))
		C.free(unsafe.Pointer(cSourcePath))
		C.free(unsafe.Pointer(cDirPath))
		C.free(unsafe.Pointer(cOutBin))
		C.free(unsafe.Pointer(cOutObj))
		C.free(unsafe.Pointer(cBuildType))
		C.free(unsafe.Pointer(cTarget))
		C.free(unsafe.Pointer(cToolchain))
		C.free(unsafe.Pointer(cMode))
		C.free(unsafe.Pointer(cCcFlags))
		C.free(unsafe.Pointer(cLdFlags))
		C.free(unsafe.Pointer(cFormat))
		C.free(unsafe.Pointer(cIsolation))
	}()

	var cctx C.fz_context_t
	cctx.plugin_path = cPluginPath
	cctx.config_path = cConfigPath
	cctx.source_path = cSourcePath
	cctx.dir_path = cDirPath
	cctx.out_bin = cOutBin
	cctx.out_obj = cOutObj
	cctx.build_type = cBuildType
	cctx.target = cTarget
	cctx.toolchain = cToolchain
	cctx.mode = cMode
	cctx.cc_flags = cCcFlags
	cctx.ld_flags = cLdFlags
	cctx.format = cFormat
	cctx.isolation = cIsolation

	var cSourceDirs []*C.char
	if goCtx.DirPath != "" {
		cSourceDirs = append(cSourceDirs, cDirPath)
	}
	for _, d := range goCtx.SourceDirs {
		cStr := C.CString(d)
		cSourceDirs = append(cSourceDirs, cStr)
		defer C.free(unsafe.Pointer(cStr))
	}

	if len(cSourceDirs) > 0 {
		cctx.source_dir_count = C.int(len(cSourceDirs))
		cctx.source_dirs = (**C.char)(C.malloc(C.size_t(len(cSourceDirs)) * C.size_t(unsafe.Sizeof(uintptr(0)))))
		arr := (*[1 << 30]*C.char)(unsafe.Pointer(cctx.source_dirs))
		copy(arr[0:len(cSourceDirs)], cSourceDirs)
		defer C.free(unsafe.Pointer(cctx.source_dirs))
	}

	p, err := m.Lookup("fz_init_module")
	if err != nil {
		return err
	}
	C.call_entry(p, unsafe.Pointer(&cctx))
	return nil
}

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
