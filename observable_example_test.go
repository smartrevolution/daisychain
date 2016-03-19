package daisychain

import "fmt"

func Example() {
	//Create an Observable that emits 0..9 ints
	o1 := Create(
		Just(0, 1, 2, 3, 4, 5, 6, 7, 8, 9),
	)

	//Create another Observable by composition
	o2 := Create(
		// o1 emits 0..9!
		o1,
		// add 1 to each number
		Map(func(ev Event) Event {
			return ev.(int) + 1
		}),
		// skip 1..5
		Skip(5),
		// take 6,7,8
		Take(3),
		// filter even numbers: 6,8
		Filter(func(ev Event) bool {
			return ev.(int)%2 == 0
		}),
		// sum of 6 and 8
		Reduce(func(ev1, ev2 Event) Event {
			return ev1.(int) + ev2.(int)
		}, 0),
	)

	//Subscribe to an Observable to execute it
	SubscribeAndWait(o2, nil /*onNext*/, nil /*onError*/, onComplete)

	//Output:
	// 14
}

func onComplete(ev Event) {
	fmt.Println(ev)
}
