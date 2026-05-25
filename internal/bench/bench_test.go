package bench

import (
	"strings"
	"testing"
)

func TestTimerJSON(t *testing.T) {
	timer := NewTimer()
	_ = timer.Stage("check", func() error { return nil })
	data, err := timer.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "total_ns") {
		t.Fatalf("unexpected json: %s", string(data))
	}
}
