//go:build ignore

package main

import (
	"fmt"

	"fz/internal/assembler"
)

func main() {
	src := []byte("start:\n    db 0x90\n    global foo\nfoo:\n    db 0xCC\n")
	out, err := assembler.EmitSourceObject(src, assembler.TargetProfileFromTarget("x86_64-linux-gnu"))
	if err != nil {
		fmt.Printf("err=%v\n", err)
		return
	}
	fmt.Printf("len=%d\n", len(out))
	for i := 0; i < len(out) && i < 64; i++ {
		fmt.Printf("%02x ", out[i])
	}
	fmt.Printf("\n")
}

