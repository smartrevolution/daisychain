package stream_test

import (
	"github.com/smartrevolution/daisychain/stream"

	"fmt"
	"sync"
	"time"
)

func Example() {
	//GIVEN
	s0 := stream.New()

	s1 := s0.Map(func(ev stream.Event) stream.Event {
		return ev.(int) * 2
	})

	s2 := s1.Reduce(func(left, right stream.Event) stream.Event {
		return left.(int) + right.(int)
	}, 0)

	s3 := s2.Filter(func(ev stream.Event) bool {
		return ev.(int) > 50
	})

	n0 := s0.Hold(-1)
	n1 := s1.Hold(-1)
	n2 := s2.Hold(-1)
	n3 := s3.Hold(-1)
	n4 := s3.Accu()

	keyfn := func(ev stream.Event) string {
		if ev.(int)%2 == 0 {
			return "even"
		}
		return "odd"
	}

	n5 := s1.Group(keyfn)

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
	fmt.Println(n4.Value()) //list of all n3 events
	fmt.Println(n5.Value()) //map of even/odd events of n0

	//Output:
	// 9
	// 18
	// 90
	// 90
	// [56 72 90]
	//map[even:[0 2 4 6 8 10 12 14 16 18]]
}
