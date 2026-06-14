/*
(c) AlexVoste
Package assembler — hot-path zero-allocation memory copy
*/

package assembler

import "unsafe"

func FastCopy(dst, src unsafe.Pointer, n uintptr) {
	for n >= 64 {
		*(*uint64)(unsafe.Pointer(uintptr(dst))) = *(*uint64)(unsafe.Pointer(uintptr(src)))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 8)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 8))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 16)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 16))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 24)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 24))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 32)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 32))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 40)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 40))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 48)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 48))
		*(*uint64)(unsafe.Pointer(uintptr(dst) + 56)) = *(*uint64)(unsafe.Pointer(uintptr(src) + 56))

		dst = unsafe.Pointer(uintptr(dst) + 64)
		src = unsafe.Pointer(uintptr(src) + 64)
		n -= 64
	}
	for n >= 8 {
		*(*uint64)(dst) = *(*uint64)(src)
		dst = unsafe.Pointer(uintptr(dst) + 8)
		src = unsafe.Pointer(uintptr(src) + 8)
		n -= 8
	}
	for n > 0 {
		*(*byte)(dst) = *(*byte)(src)
		dst = unsafe.Pointer(uintptr(dst) + 1)
		src = unsafe.Pointer(uintptr(src) + 1)
		n--
	}
}

func Copy(dst, src unsafe.Pointer, n int) {
	if n <= 0 {
		return
	}
	FastCopy(dst, src, uintptr(n))
}
