package bench

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Stage struct {
	Name       string `json:"name"`
	DurationNs int64  `json:"duration_ns"`
}

type Timer struct {
	stages []Stage
	total  int64
	err    error
	mu     sync.Mutex
}

func NewTimer() *Timer {
	return &Timer{}
}

func (t *Timer) Stage(name string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start).Nanoseconds()
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stages = append(t.stages, Stage{Name: name, DurationNs: duration})
	t.total += duration
	if err != nil && t.err == nil {
		t.err = err
	}
	return err
}

func (t *Timer) Error() error {
	return t.err
}

func (t *Timer) Report() string {
	var b strings.Builder
	for _, stage := range t.stages {
		b.WriteString(fmt.Sprintf("%s: %d ns (%d ms)\n", stage.Name, stage.DurationNs, stage.DurationNs/1e6))
	}
	b.WriteString(fmt.Sprintf("total: %d ns (%d ms)\n", t.total, t.total/1e6))
	return b.String()
}

func (t *Timer) JSON() ([]byte, error) {
	report := struct {
		Stages  []Stage `json:"stages"`
		TotalNs int64   `json:"total_ns"`
	}{
		Stages:  t.stages,
		TotalNs: t.total,
	}
	return json.Marshal(report)
}
