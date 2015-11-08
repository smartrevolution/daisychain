package streams

import (
	"fmt"
	"sync"
	"time"
)

func ExampleStream() {
	s0 := NewStream()
	numbers := s0.Hold()

	var wg sync.WaitGroup
	wg.Add(1)
	func() {
		for i := 0; i < 10; i++ {
			s0.Send(i)
		}
		wg.Done()
	}()
	wg.Wait()
	time.Sleep(100 * time.Millisecond)
	fmt.Println(numbers.Events())

	//Output:
	//[0 1 2 3 4 5 6 7 8 9]

}
