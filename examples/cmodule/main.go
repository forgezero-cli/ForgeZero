package main

import (
	"fmt"
	"unsafe"

	"fz/internal/cplugin"
)

func main() {
	m, err := cplugin.Load("./libfz_example.so")
	if err != nil {
		panic(err)
	}
	defer m.Close()
	var v int64 = 41
	if err := m.CallModuleEntry("fz_module", unsafe.Pointer(&v)); err != nil {
		panic(err)
	}
	fmt.Println(v)
}
