package main

import (
	"fmt"
	"sync"
)

type Counter struct {
	mu    sync.Mutex
	value int
}

func (c *Counter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

func (c *Counter) Value() {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Println(c.value)
}

func main() {
	const goroutines = 5
	const incrementsPerGoroutine = 2
	counter := Counter{
		mu:    sync.Mutex{},
		value: 0,
	}

	var wg sync.WaitGroup

	for range goroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for range incrementsPerGoroutine {
				counter.Increment()
				counter.Value()
			}

		}()
	}

	wg.Wait()

}
