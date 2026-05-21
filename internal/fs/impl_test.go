//go:build !windows

package fs

import "testing"

func TestImplNameUnix(t *testing.T) {
	if ImplName() != "unix" {
		t.Fatalf("got %q", ImplName())
	}
}
