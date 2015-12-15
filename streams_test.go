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
	t.Parallel()
	//GIVEN
	parent := newStream()
	child := newStream()
	time.Sleep(50 * time.Millisecond)

	//WHEN
	parent.subscribe(child)
	time.Sleep(50 * time.Millisecond)
	parent.close()

	//THEN
	if l := len(parent.subs); l != 0 {
		t.Error("Expected 0, Got:", l)
	}
}

func TestSubscribers(t *testing.T) {
	t.Parallel()
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

func TestVars(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := NewSink()
	mapped := sink.Map(func(s *Stream, ev Event) Event {
		s.SetVar("count", s.Var("count", 0).(int)+1)
		return ev
	})
	signal := mapped.Hold(0)

	//WHEN
	send0to9(sink)

	//THEN
	if val := signal.Value(); val != 9 {
		t.Error("Expected: 9, Got:", val)
	}
	if count := sink.Var("count", 666); count != 10 {
		t.Error("Expected: 10, Got:", count)
	}
}

func TestSink(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := NewSink()
	signal := sink.Hold(0)

	//WHEN
	send0to9(sink)

	//THEN
	if val := signal.Value(); val != 9 {
		t.Error("Expected: 9, Got:", val)
	}
}

func TestMap(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := NewSink()

	squared := sink.Map(func(s *Stream, ev Event) Event {
		return ev.(int) * ev.(int)
	})

	signal := squared.Hold(0)

	//WHEN
	send0to9(sink)

	//THEN
	if val := signal.Value(); val != 81 {
		t.Error("Expected: 81, Got:", val)
	}
}

func TestReduce(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := NewSink()

	squared := sink.Reduce(func(a, b Event) Event {
		return a.(int) + b.(int)
	}, 100)

	signal := squared.Hold(0)

	//WHEN
	send0to9(sink)

	//THEN
	if val := signal.Value(); val != 145 {
		t.Error("Expected: 145, Got:", val)
	}
}

func TestFilter(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := NewSink()

	evenNums := sink.Filter(func(ev Event) bool {
		return ev.(int)%2 == 0
	})

	signal := evenNums.Hold(0)

	//WHEN
	send0to9(sink)

	//THEN
	if val := signal.Value(); val != 8 {
		t.Error("Expected: 8, Got:", val)
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()

	//GIVEN
	s0 := NewSink()
	s1 := NewSink()

	map0 := s0.Map(func(s *Stream, ev Event) Event {
		return ev
	})
	map1 := s1.Map(func(s *Stream, ev Event) Event {
		return ev
	})

	merged := map0.Merge(map1)
	added := merged.Reduce(func(a, b Event) Event {
		return a.(int) + b.(int)
	}, 0)

	//WHEN
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

	//THEN
	if last := sum.Value(); last != 10 {
		t.Error("Expected: 10, Got:", last)
	}
}

func TestThrottle(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := NewSink()
	mapped := sink.Map(func(s *Stream, ev Event) Event {
		return ev
	})
	throttled := mapped.Throttle(10 * time.Millisecond)

	signal := throttled.Hold(666)

	//THEN
	signal.OnValue(func(ev Event) {
		expected := []Event{1, 1, 1}
		if got, ok := ev.([]Event); !ok || len(got) != 3 {
			t.Errorf("Expected %#v, Got: %#v", expected, got)
		}
	})

	//WHEN
	sink.Send(1)
	sink.Send(1)
	sink.Send(1)
	time.Sleep(15 * time.Millisecond)
}

func TestCondition(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := NewSink()
	mapped := sink.Map(func(s *Stream, ev Event) Event {
		return ev
	})

	//THEN
	_ = mapped.Condition(func(ev Event) bool {
		return ev.(int) == 2
	}, func(ev Event) {
		if ev.(int) != 2 {
			t.Error("Expected: 2, Got:", ev)
		}
	})

	//WHEN
	sink.Send(1)
	sink.Send(2)
	sink.Send(3)
	time.Sleep(15 * time.Millisecond)
}

func TestAccu(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := NewSink()
	signal := sink.Accu()

	//WHEN
	send0to9(sink)

	//THEN
	expected := []Event{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if val := signal.Value().([]Event); len(val) != len(expected) {
		t.Errorf("Expected: %#v, Got: %#v", expected, val)
	}
}

func TestGroup(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := NewSink()
	signal := sink.Group(func(ev Event) string {
		if ev.(int)%2 == 0 {
			return "even"
		}
		return "odd"
	})

	//WHEN
	send0to9(sink)

	//THEN
	if evenOdd, ok := signal.Value().(group); !ok || len(evenOdd) != 2 {
		t.Log(signal.Value())
		t.Errorf("Expected: map[even:[0 2 4 6 8] odd:[1 3 5 7 9]], Got: %#v", evenOdd)
	}
}

func TestErrorHandling(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := NewSink()

	//WHEN
	signal := sink.Hold(666)

	//THEN
	if val := signal.Value(); val != 666 {
		t.Error("Expected: 666, Got:", val)
	}

	//WHEN
	sink.Send(1)
	time.Sleep(10 * time.Millisecond)

	//THEN
	if val := signal.Value(); val != 1 {
		t.Error("Expected: 1, Got:", val)
	}

	//WHEN
	empty := NewEmptyEvent()
	sink.Send(empty)
	time.Sleep(10 * time.Millisecond)

	//THEN
	if val, ok := signal.Value().(Empty); ok && val != empty {
		t.Errorf("Expected: %#v, Got: %#v", empty, val)
	}

	//WHEN
	sink.Send(NewErrorEvent("errormsg"))
	time.Sleep(10 * time.Millisecond)

	//THEN
	if val, ok := signal.Value().(Error); !ok {
		t.Error("Expected: anError, Got:", val)
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
	t.Parallel()

	//GIVEN
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

	//WHEN
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

	//THEN
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
