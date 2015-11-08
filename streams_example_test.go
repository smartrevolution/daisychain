package streams

import (
	"fmt"
	"sync"
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

	//THEN
	fmt.Println(s0.Events()) //stream
	fmt.Println(s1.Events()) //map
	fmt.Println(s2.Events()) //reduce
	fmt.Println(s3.Events()) //filter
	s0.Close()

	//Output:
	//[0 1 2 3 4 5 6 7 8 9]
	//[0 2 4 6 8 10 12 14 16 18]
	//[0 2 6 12 20 30 42 56 72 90]
	//[56 72 90]

}
