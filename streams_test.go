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

func ExampleStream_Update() {
	sig, s0 := numbers()
	fmt.Println(sig.Events())

	var wg sync.WaitGroup
	sig.OnValue(func(ev Event) {
		wg.Done()
	})
	wg.Add(1)
	s0.Update(10)
	wg.Wait()
	fmt.Println(sig.Events())

	wg.Add(1)
	s0.Update(11)
	wg.Wait()
	fmt.Println(sig.Events())

	wg.Add(1)
	s0.Send(12)
	wg.Wait()
	fmt.Println(sig.Events())

	//Output:
	//[0 1 2 3 4 5 6 7 8 9]
	//[1 2 3 4 5 6 7 8 9 10]
	//[2 3 4 5 6 7 8 9 10 11]
	//[2 3 4 5 6 7 8 9 10 11 12]
}

func numbers() (*Signal, *Sink) {
	s0 := NewSink()
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

	return numbers, s0
}
