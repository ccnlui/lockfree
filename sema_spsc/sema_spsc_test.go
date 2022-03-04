package sema_spsc

import (
	"fmt"
	"testing"
	"time"
)

func TestSemaSPSC(t *testing.T) {
	q := NewRingBuffer(16)

	// Consumer.
	go func() {
		for {
			got, _ := q.Get()
			fmt.Println("got", got)
			time.Sleep(time.Millisecond)
		}
	}()

	// Producer.
	ticker := time.NewTicker(3 * time.Second)
	for i := 0; i < 1_000; i++ {
		select {
		case <-ticker.C:
			// fmt.Println("producer pauses...")
			// time.Sleep(time.Second)
		default:
		}
		q.Put(i)
		fmt.Println("put", i)
	}
}

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

func BenchmarkDSPSC(b *testing.B) {
	q := NewRingBuffer(8192)

	b.ResetTimer()
	go func() {
		for i := 0; i < b.N; i++ {
			// fmt.Println("read", i)
			_, _ = q.Get()
			// fmt.Printf("got %v at %v\n", got, i)
		}
	}()

	for i := 0; i < b.N; i++ {
		// fmt.Println("write", i)
		_ = q.Put(`a`)
		// fmt.Printf("put %v at %v\n", err, i)

	}
}
