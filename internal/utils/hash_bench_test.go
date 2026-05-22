package utils

import (
	"os"
	"testing"
)

func BenchmarkHashFile100MB(b *testing.B) {
	tmpFile, err := os.CreateTemp("", "bench100")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	data := make([]byte, 100*1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	if _, err := tmpFile.Write(data); err != nil {
		b.Fatal(err)
	}
	tmpFile.Close()

	filePath := tmpFile.Name()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := hashRawFileDigest(filePath)
		if err != nil {
			b.Fatal(err)
		}
	}
}
