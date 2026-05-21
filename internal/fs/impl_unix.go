//go:build !windows

package fs

func ImplName() string {
	return "unix"
}
