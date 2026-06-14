//go:build amd64
// +build amd64

/*
(c) AlexVoste
Package utils — raw assembly calls for maximum performance
*/

package utils

import "unsafe"

//go:noescape
func callRaw2(code unsafe.Pointer, p unsafe.Pointer, n uintptr)
