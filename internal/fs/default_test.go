//go:build !windows

package fs

import "testing"

func TestDefaultIsUnix(t *testing.T) {
	if _, ok := Default.(Unix); !ok {
		t.Fatalf("Default type %T", Default)
	}
}
