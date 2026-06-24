package main

import (
	"fmt"
	"unsafe"

	"github.com/forgezero-cli/ForgeZero/internal/cplugin"
)

func main() {
	m, err := cplugin.Load("./libfz_example.so")
	if err != nil {
		panic(err)
	}
	defer m.Close()
	var ctxValue int64 = 41
	if err := m.CallEntryBySymbol("fz_init_module", unsafe.Pointer(&ctxValue)); err != nil {
		panic(err)
	}
	fmt.Println(ctxValue)
}
