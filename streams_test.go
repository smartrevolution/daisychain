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
		s0 := NewStream()
		t.Log(s0)
		time.Sleep(1 * time.Second)
		runtime.GC()
	}
}

func TestFind(t *testing.T) {
	nums := numbers()

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
	nums := numbers()
	fmt.Println(nums.Events())

	//Output:
	//[0 1 2 3 4 5 6 7 8 9]
}

func numbers() *Signal {
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

	return numbers
}
