package collect

import (
	"testing"
	"time"
)

func BenchmarkSnapshot(b *testing.B) {
	c := New()
	// warm up: prev-snapshot deltas
	if _, err := c.Snapshot(); err != nil {
		b.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.Snapshot(); err != nil {
			b.Fatal(err)
		}
	}
}
