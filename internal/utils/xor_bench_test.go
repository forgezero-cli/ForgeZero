package utils

import (
	"testing"
)

func BenchmarkExecRawXor1MB(b *testing.B) {
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExecRawXor(data)
	}
}
