package daisychain

import (
	"fmt"
	"sync"
	"time"
)

func Example() {
	//GIVEN
	s0 := NewSink()

	s1 := s0.Map(func(s *Stream, ev Event) Event {
		return ev.(int) * 2
	})

	s2 := s1.Reduce(func(s *Stream, left, right Event) Event {
		return left.(int) + right.(int)
	}, 0)

	s3 := s2.Filter(func(s *Stream, ev Event) bool {
		return ev.(int) > 50
	})

	n0 := s0.Hold(-1)
	n1 := s1.Hold(-1)
	n2 := s2.Hold(-1)
	n3 := s3.Hold(-1)

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
	fmt.Println(n0.Value()) //stream = 0..9
	fmt.Println(n1.Value()) //map = 9 * 2 = 18
	fmt.Println(n2.Value()) //reduce = sum(0..9)
	fmt.Println(n3.Value()) //filter = max(sum(0..9)), when > 50

	//Output:
	// 9
	// 18
	// 90
	// 90

}
