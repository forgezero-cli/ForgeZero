//go:build !windows

package fs

var Default FileSystem = Unix{}
