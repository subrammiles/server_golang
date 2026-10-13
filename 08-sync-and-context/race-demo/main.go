// A DATA RACE on purpose. Run it twice and see different (wrong) totals, then let Go find the bug:
//
//	go run ./08-sync-and-context/race-demo
//	go run -race ./08-sync-and-context/race-demo     -> "WARNING: DATA RACE"
package main

import (
	"fmt"
	"sync"
)

func main() {
	counter := 0
	var wg sync.WaitGroup
	for range 1000 {
		wg.Go(func() {
			counter++ // read-modify-write from many goroutines with NO synchronisation
		})
	}
	wg.Wait()
	fmt.Println("counter =", counter, "(expected 1000)")
}
