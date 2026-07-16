//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/forgezero-cli/ForgeZero/internal/builder"
)

func main() {
    incs, cflags, ldflags := builder.DiscoverMakefileSettings("nginx")
    fmt.Printf("incs=%q\ncflags=%q\nldflags=%q\n", incs, cflags, ldflags)
}
