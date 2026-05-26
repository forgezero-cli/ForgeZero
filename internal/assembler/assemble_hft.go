package assembler

import "unsafe"

func FastCopy(dst, src unsafe.Pointer, n uintptr) {
    d := uintptr(dst)
    s := uintptr(src)
    for n >= 64 {
        *(*uint64)(unsafe.Pointer(d+0)) = *(*uint64)(unsafe.Pointer(s+0))
        *(*uint64)(unsafe.Pointer(d+8)) = *(*uint64)(unsafe.Pointer(s+8))
        *(*uint64)(unsafe.Pointer(d+16)) = *(*uint64)(unsafe.Pointer(s+16))
        *(*uint64)(unsafe.Pointer(d+24)) = *(*uint64)(unsafe.Pointer(s+24))
        *(*uint64)(unsafe.Pointer(d+32)) = *(*uint64)(unsafe.Pointer(s+32))
        *(*uint64)(unsafe.Pointer(d+40)) = *(*uint64)(unsafe.Pointer(s+40))
        *(*uint64)(unsafe.Pointer(d+48)) = *(*uint64)(unsafe.Pointer(s+48))
        *(*uint64)(unsafe.Pointer(d+56)) = *(*uint64)(unsafe.Pointer(s+56))
        d += 64
        s += 64
        n -= 64
    }
    for n >= 8 {
        *(*uint64)(unsafe.Pointer(d)) = *(*uint64)(unsafe.Pointer(s))
        d += 8
        s += 8
        n -= 8
    }
    for n > 0 {
        *(*byte)(unsafe.Pointer(d)) = *(*byte)(unsafe.Pointer(s))
        d++
        s++
        n--
    }
}

func Copy(dst, src unsafe.Pointer, n int) {
    if n <= 0 {
        return
    }
    FastCopy(dst, src, uintptr(n))
}
