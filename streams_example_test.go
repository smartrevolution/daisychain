package streams

import (
	"fmt"
	"sync"
	"time"
)

func Example() {
	//GIVEN
	s0 := NewStream()

	s1 := s0.Map(func(ev Event) Event {
		return ev.(int) * 2
	})

	s2 := s1.Reduce(func(left, right Event) Event {
		return left.(int) + right.(int)
	}, 0)

	s3 := s2.Filter(func(ev Event) bool {
		return ev.(int) > 50
	})

	n0 := s0.Hold()
	n1 := s1.Hold()
	n2 := s2.Hold()
	n3 := s3.Hold()

	//WHEN
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

	//THEN
	fmt.Println(n0.Events()) //stream
	fmt.Println(n1.Events()) //map
	fmt.Println(n2.Events()) //reduce
	fmt.Println(n3.Events()) //filter
	s0.Close()

	//Output:
	//[0 1 2 3 4 5 6 7 8 9]
	//[0 2 4 6 8 10 12 14 16 18]
	//[0 2 6 12 20 30 42 56 72 90]
	//[56 72 90]

}
