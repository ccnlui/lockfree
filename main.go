package main

import (
	"fmt"
	"lockfree/cspsc"
	"sync"
)

func main() {
	fmt.Println("lockfree!")
	var wg sync.WaitGroup

	rb := cspsc.NewRingBuffer(8192)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			v, err := rb.Get()
			if err != nil {
				fmt.Println("queue closed", err)
				return
			}
			fmt.Println("recv:", v)
		}
	}()
	for i := 0; i < 5; i++ {
		err := rb.Put(42)
		if err != nil {
			fmt.Println("put failed", err)
		}
	}
	wg.Wait()
}
