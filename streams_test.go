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

func xTestClose(t *testing.T) {
	//GIVEN
	parent := newStream()
	child := newStream()

	//WHEN
	parent.subscribe(child)
	parent.close()

	//THEN
	if l := len(parent.subs); l != 0 {
		t.Error("Expected 0, Got:", l)
	}
}

func TestSubscribers(t *testing.T) {
	//GIVEN
	parent := newStream()
	child := newStream()

	//WHEN
	parent.subscribe(child)

	//THEN
	if l := len(parent.subs); l != 1 {
		t.Error("Expected 1, Got:", l)
	}

	//WHEN
	parent.unsubscribe(child)

	//THEN
	if l := len(parent.subs); l != 0 {
		t.Error("Expected 0, Got:", l)
	}
}

func TestSink(t *testing.T) {
	sink := NewSink()
	signal := sink.Hold(0)

	send0to9(sink)

	if val := signal.Value(); val != 9 {
		t.Error("Expected: 9, Got:", val)
	}
}

func TestMap(t *testing.T) {
	sink := NewSink()

	squared := sink.Map(func(ev Event) Event {
		return ev.(int) * ev.(int)
	})

	signal := squared.Hold(0)

	send0to9(sink)

	if val := signal.Value(); val != 81 {
		t.Error("Expected: 81, Got:", val)
	}
}

func TestReduce(t *testing.T) {
	sink := NewSink()

	squared := sink.Reduce(func(a, b Event) Event {
		return a.(int) + b.(int)
	}, 100)

	signal := squared.Hold(0)

	send0to9(sink)

	if val := signal.Value(); val != 145 {
		t.Error("Expected: 145, Got:", val)
	}
}

func TestFilter(t *testing.T) {
	sink := NewSink()

	evenNums := sink.Filter(func(ev Event) bool {
		return ev.(int)%2 == 0
	})

	signal := evenNums.Hold(0)

	send0to9(sink)

	if val := signal.Value(); val != 8 {
		t.Error("Expected: 8, Got:", val)
	}
}

func TestMerge(t *testing.T) {
	s0 := NewSink()
	s1 := NewSink()

	map0 := s0.Map(func(ev Event) Event {
		return ev
	})
	map1 := s1.Map(func(ev Event) Event {
		return ev
	})

	merged := map0.Merge(map1)
	added := merged.Reduce(func(a, b Event) Event {
		return a.(int) + b.(int)
	}, 0)

	var wg sync.WaitGroup

	sig := merged.Hold(0)
	sig.OnValue(func(ev Event) {
		wg.Done()
	})

	sum := added.Hold(0)

	wg.Add(4)
	s0.Send(1)
	s1.Send(2)
	s1.Send(3)
	s0.Send(4)
	wg.Wait()

	time.Sleep(10 * time.Millisecond)

	if last := sum.Value(); last != 10 {
		t.Error("Expected: 10, Got:", last)
	}
}

type estimation struct {
	key string
	min int
	max int
}

func (e estimation) String() string {
	return fmt.Sprintf("%d-%d", e.min, e.max)
}

func TestReduceWithStruct(t *testing.T) {
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

	wg.Add(3)
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
	estimations.Send(estimation{
		key: "k3",
		min: 3,
		max: 4,
	})
	wg.Wait()
	time.Sleep(50 * time.Millisecond)
	if est := sum.Value().(estimation); est.min != 6 && est.max != 9 {
		t.Error("Expected: 6-9, Got:", est)
	}
}

func send0to9(s *Sink) {
	var wg sync.WaitGroup
	wg.Add(1)
	func() {
		for i := 0; i < 10; i++ {
			s.Send(i)
		}
		wg.Done()
	}()
	wg.Wait()
	time.Sleep(10 * time.Millisecond)
}

func numbers() (*Signal, *Sink) {
	s0 := NewSink()
	numbers := s0.Hold(0)

	var wg sync.WaitGroup
	wg.Add(1)
	func() {
		for i := 0; i < 10; i++ {
			s0.Send(i)
		}
		wg.Done()
	}()
	wg.Wait()
	time.Sleep(10 * time.Millisecond)

	return numbers, s0
}
