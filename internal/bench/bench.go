package bench

import (
	"encoding/json"
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
	var buf []byte
	buf = append(buf, "stages:\n"...)
	for _, s := range t.stages {
		buf = append(buf, s.Name...)
		buf = append(buf, ": "...)
		buf = appendInt64(buf, s.DurationNs)
		buf = append(buf, " ns ("...)
		buf = appendInt64(buf, s.DurationNs/1e6)
		buf = append(buf, " ms)\n"...)
	}
	buf = append(buf, "total: "...)
	buf = appendInt64(buf, t.total)
	buf = append(buf, " ns ("...)
	buf = appendInt64(buf, t.total/1e6)
	buf = append(buf, " ms)\n"...)
	return string(buf)
}

func appendInt64(b []byte, v int64) []byte {
	if v == 0 {
		return append(b, '0')
	}
	var sign byte
	if v < 0 {
		sign = '-'
		v = -v
	}
	var tmp [20]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = byte('0' + v%10)
		v /= 10
	}
	if sign != 0 {
		i--
		tmp[i] = sign
	}
	return append(b, tmp[i:]...)
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