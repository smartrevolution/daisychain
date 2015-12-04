package daisychain

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func xTestFinalizer(t *testing.T) {
	for i := 0; i < 3; i++ {
		s0 := NewSink()
		t.Log(s0)
		time.Sleep(1 * time.Second)
		runtime.GC()
	}
}

func TestUpdateableSink(t *testing.T) {
	//GIVEN
	sink := NewSink()
	nums := sink.Hold(0)

	add := sink.Reduce(func(left, right Event) Event {
		if y, ok := right.(int); ok {
			return left.(int) + y
		}
		return 0
	}, 0)

	//WHEN
	sum := add.Hold(0)

	var wg sync.WaitGroup
	sum.OnValue(func(ev Event) {
		wg.Done()
	})

	//THEN
	if last := sum.Last(); last != 0 {
		t.Error("Expected: 0, Got:", last)
	}

	//WHEN
	wg.Add(1)
	sink.Send(1)
	wg.Wait()

	//THEN
	if last := sum.Last(); last != 1 {
		t.Error("Expected: 1, Got:", last)
	}

	//WHEN
	wg.Add(2)
	sink.Send(2)
	sink.Send(3)
	wg.Wait()

	t.Log(nums.Events())

	//THEN
	if last := sum.Last(); last != 6 {
		t.Error("Expected: 6, Got:", last)
	}

	//WHEN
	wg.Add(1)
	sink.Update(4, func(ev Event) bool {
		return ev.(int) == 3
	})
	wg.Wait()

	//THEN
	if sum.Last() != 7 {
		t.Error(sum.Events())
	}
}

func TestFind(t *testing.T) {
	nums, _ := numbers()

	six, ok := nums.Get(func(ev Event) bool {
		if ev.(int) == 6 {
			return true
		}
		return false
	})

	if !ok {
		t.Error(ok, six)
	}

	if six.(int) != 6 {
		t.Error(six)
	}
}

func ExampleStream() {
	nums, _ := numbers()
	fmt.Println(nums.Events())

	//Output:
	//[0 1 2 3 4 5 6 7 8 9]
}

func ExampleStream_Update_int() {
	sig, s0 := numbers()
	fmt.Println(sig.Events())

	var wg sync.WaitGroup
	sig.OnValue(func(ev Event) {
		wg.Done()
	})
	wg.Add(1)
	s0.Update(10, func(ev Event) bool {
		if num, ok := ev.(int); ok {
			if num == 0 {
				return true
			}
		}
		return false
	})
	wg.Wait()
	fmt.Println(sig.Events())

	wg.Add(1)
	s0.Update(11, func(ev Event) bool {
		if num, ok := ev.(int); ok {
			if num == 5 {
				return true
			}
		}
		return false
	})
	wg.Wait()
	fmt.Println(sig.Events())

	wg.Add(1)
	s0.Send(12)
	wg.Wait()
	fmt.Println(sig.Events())

	//Output:
	//[0 1 2 3 4 5 6 7 8 9]
	//[1 2 3 4 5 6 7 8 9 10]
	//[1 2 3 4 6 7 8 9 10 11]
	//[1 2 3 4 6 7 8 9 10 11 12]
}

type estimation struct {
	key string
	min int
	max int
}

func (e estimation) String() string {
	return fmt.Sprintf("%d-%d", e.min, e.max)
}

func ExampleStream_Update_struct() {
	empty := estimation{
		key: "k0",
	}
	estimations := NewSink()
	add := estimations.Reduce(func(a, b Event) Event {
		return estimation{
			min: a.(estimation).min + b.(estimation).min,
			max: a.(estimation).max + b.(estimation).max,
		}
	}, empty)
	sum := add.Hold(empty)

	var wg sync.WaitGroup
	sum.OnValue(func(ev Event) {
		wg.Done()
	})

	wg.Add(2)
	estimations.Send(estimation{
		key: "k1",
		min: 1,
		max: 2,
	})
	estimations.Send(estimation{
		key: "k2",
		min: 2,
		max: 3,
	})
	wg.Wait()
	fmt.Println(sum.Last())

	wg.Add(1)
	estimations.Update(estimation{
		key: "k1",
		min: 3,
		max: 4,
	}, func(ev Event) bool {
		if est, ok := ev.(estimation); ok && est.key == "k1" {
			return true
		}
		return false
	})
	wg.Wait()
	fmt.Println(sum.Last())

	// s0.Update(10, func(ev Event) bool {
	// 	if num, ok := ev.(int); ok {
	// 		if num == 0 {
	// 			return true
	// 		}
	// 	}
	// 	return false
	// })
	// wg.Wait()
	// fmt.Println(sig.Events())

	// wg.Add(1)
	// s0.Update(11, func(ev Event) bool {
	// 	if num, ok := ev.(int); ok {
	// 		if num == 5 {
	// 			return true
	// 		}
	// 	}
	// 	return false
	// })
	// wg.Wait()
	// fmt.Println(sig.Events())

	// wg.Add(1)
	// s0.Send(12)
	// wg.Wait()
	// fmt.Println(sig.Events())

	//Output:
	//3-5
	//5-7
}

func numbers() (*Signal, *Sink) {
	s0 := NewSink()
	numbers := s0.Hold(-1)

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

	return numbers, s0
}
