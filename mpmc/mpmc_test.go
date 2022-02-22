package mpmc

import "testing"

func BenchmarkChannel(b *testing.B) {
	ch := make(chan interface{}, 8192)

	b.ResetTimer()
	go func() {
		for i := 0; i < b.N; i++ {
			<-ch
		}
	}()

	for i := 0; i < b.N; i++ {
		ch <- `a`
	}
}

func BenchmarkMPMC(b *testing.B) {
	q := NewRingBuffer(8192)

	b.ResetTimer()
	go func() {
		for i := 0; i < b.N; i++ {
			q.Get()
		}
	}()

	for i := 0; i < b.N; i++ {
		q.Put(`a`)
	}
}
