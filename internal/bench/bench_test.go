package bench

import (
	"strings"
	"testing"
	"time"
)

func TestTimerRecordsStages(t *testing.T) {
	timer := NewTimer()
	if err := timer.Stage("step1", func() error {
		time.Sleep(1 * time.Millisecond)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := timer.Stage("step2", func() error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if timer.Error() != nil {
		t.Fatal("expected no error")
	}
	report := timer.Report()
	if !strings.Contains(report, "step1") || !strings.Contains(report, "step2") {
		t.Fatalf("unexpected report: %s", report)
	}
	if timer.total <= 0 {
		t.Fatal("expected total duration recorded")
	}
}

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
