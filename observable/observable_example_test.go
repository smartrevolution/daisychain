package observable_test

import (
	"github.com/smartrevolution/daisychain/observable"

	"fmt"
	"time"
)

func Example() {
	//GIVEN
	s0 := observable.New()
	defer s0.Close()

	s1 := s0.Map(func(ev observable.Event) observable.Event {
		return ev.(int) * 2
	})

	s2 := s1.Reduce(func(left, right observable.Event) observable.Event {
		return left.(int) + right.(int)
	}, 0)

	s3 := s2.Filter(func(ev observable.Event) bool {
		return ev.(int) > 50
	})

	n0 := s0.Hold(0)
	n1 := s1.Hold(0)
	n2 := s2.Hold(0)
	n3 := s3.Hold(0)
	n4 := s3.Collect().Hold(observable.Empty())

	keyfn := func(ev observable.Event) string {
		if ev.(int)%2 == 0 {
			return "even"
		}
		return "odd"
	}

	n5 := s1.GroupBy(keyfn).Hold(observable.Empty())

	//WHEN
	s0.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(100 * time.Millisecond)

	//THEN
	fmt.Println(n0.Value()) //observable = 0..9
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
