package mpmc

import (
	"testing"
)

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

func BenchmarkChannelConcurrentWrite(b *testing.B) {
	ch := make(chan interface{}, 8192)
	// numGr := runtime.GOMAXPROCS(0)

	b.ResetTimer()
	// 1 Consumer.
	go func() {
		for i := 0; i < b.N; i++ {
			<-ch
		}
	}()

	// N Producers.
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch <- `a`
		}
	})
}

func BenchmarkMPMCConcurrentWrite(b *testing.B) {
	q := NewRingBuffer(8192)
	// numGr := runtime.GOMAXPROCS(0)

	b.ResetTimer()
	// 1 Consumer.
	go func() {
		for i := 0; i < b.N; i++ {
			q.Get()
		}
	}()

	// N Producers.
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Put(`a`)
		}
	})
}
