package stream

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func xTestFinalizer(t *testing.T) {
	for i := 0; i < 3; i++ {
		s0 := New()
		t.Log(s0)
		time.Sleep(1 * time.Second)
		runtime.GC()
	}
}

func TestClose(t *testing.T) {
	t.Parallel()

	sink := New()
	mapped := sink.Map(func(ev Event) Event {
		return ev //do nothing
	})

	sink.From(1, 2, 3, 4, 5)
	sink.Close()

	sink = New()
	mapped = sink.Map(func(ev Event) Event {
		return ev //do nothing
	})
	signal := mapped.Hold(0)

	sink.From(1, 2, 3, 4, 5)
	signal.Close()
	//not a really intelligent test...but...
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

func TestNew(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := New()
	signal := sink.Hold(0)

	//WHEN
	sink.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	if val := signal.Value(); val != 9 {
		t.Error("Expected: 9, Got:", val)
	}
}

func TestMap(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := New()

	squared := sink.Map(func(ev Event) Event {
		return ev.(int) * ev.(int)
	})

	signal := squared.Hold(0)

	//WHEN
	sink.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	if val := signal.Value(); val != 81 {
		t.Error("Expected: 81, Got:", val)
	}
}

func TestReduce(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := New()

	squared := sink.Reduce(func(a, b Event) Event {
		return a.(int) + b.(int)
	}, 100)

	signal := squared.Hold(0)

	//WHEN
	sink.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	if val := signal.Value(); val != 145 {
		t.Error("Expected: 145, Got:", val)
	}
}

func TestFilter(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := New()

	evenNums := sink.Filter(func(ev Event) bool {
		return ev.(int)%2 == 0
	})

	signal := evenNums.Hold(0)

	//WHEN
	sink.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	if val := signal.Value(); val != 8 {
		t.Error("Expected: 8, Got:", val)
	}
}

func TestThrottle(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := New()
	mapped := sink.Map(func(ev Event) Event {
		return ev
	})
	throttled := mapped.Throttle(10 * time.Millisecond)

	signal := throttled.Hold(666)

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

func TestMerge(t *testing.T) {
	t.Parallel()

	//GIVEN
	s0 := New()
	s1 := New()

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

func TestCollect(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := New()
	signal := sink.Collect()

	//WHEN
	sink.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	expected := []Event{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if val, ok := signal.Value().([]Event); len(val) != len(expected) || !ok {
		t.Errorf("Expected: %#v, Got: %#v", expected, val)
	}
}

func TestGroupBy(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := New()
	signal := sink.GroupBy(func(ev Event) string {
		if ev.(int)%2 == 0 {
			return "even"
		}
		return "odd"
	})

	//WHEN
	sink.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	if evenOdd, ok := signal.Value().(group); len(evenOdd) != 2 || !ok {
		t.Log(signal.Value())
		t.Errorf("Expected: map[even:[0 2 4 6 8] odd:[1 3 5 7 9]], Got: %#v", evenOdd)
	}
}

func TestDistinct(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := New()
	signal := sink.Distinct(func(ev Event) string {
		if ev.(int)%2 == 0 {
			return "even"
		}
		return "odd"
	})

	//WHEN
	sink.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	if evenOdd, ok := signal.Value().(map[string]Event); len(evenOdd) != 2 || !ok {
		t.Log(signal.Value())
		t.Errorf("Expected: map[even:8 odd:9], Got: %#v", evenOdd)
	}
}

func TestSignalOnChange(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup

	//GIVEN
	sink := New()

	//WHEN/THEN
	signal := sink.Hold(0)
	values := signal.OnValue(func(ev Event) { //1
		switch ev.(type) {
		case ErrorEvent:
			t.Error("Received error event in OnValue()-1")
		case CompleteEvent:
			t.Error("Received complete event in OnValue()-1")
		case EmptyEvent:
			//all good, but do nothing
		default: //Event (with value)
			t.Logf("OnValue=%T", ev)
			wg.Done()
		}
	})
	empty := signal.OnValue(func(ev Event) { //2
		switch ev.(type) {
		case ErrorEvent:
			t.Error("Received error event in OnValue()-2")
		case CompleteEvent:
			t.Error("Received complete event in OnValue()-2")
		case EmptyEvent:
			t.Logf("OnValue=%T", ev)
			wg.Done()
		default: //Event (with value)
			//all good, but do nothing
		}
	})
	errors := signal.OnError(func(ev Event) {
		switch ev.(type) {
		case ErrorEvent:
			t.Logf("OnError=%T", ev)
			wg.Done()
		case CompleteEvent:
			t.Error("Received complete event in OnError()")
		case EmptyEvent:
			t.Error("Received empty event in OnError()")
		default: //Event (with value)
			t.Error("Received value event in OnError()")
			//all good, but do nothing
		}
	})
	completed := signal.OnComplete(func(ev Event) {
		switch ev.(type) {
		case ErrorEvent:
			t.Error("Received complete event in OnComplete()")
		case CompleteEvent:
			t.Logf("OnComplete=%T", ev)
			wg.Done()
		case EmptyEvent:
			t.Error("Received empty event in OnComplete()")
		default: //Event (with value)
			t.Error("Received value event in OnComplete()")
			//all good, but do nothing
		}
	})

	wg.Add(4)
	sink.Send(1)
	sink.Send(Empty())
	sink.Send(Error("Testerror"))
	sink.Send(Complete())
	time.Sleep(10 * time.Millisecond)
	wg.Wait()

	//THEN
	if v := len(signal.values); v != 2 {
		t.Error("Expected 2, Got:", v)
	}
	if e := len(signal.errors); e != 1 {
		t.Error("Expected 1, Got:", e)
	}
	if c := len(signal.completed); c != 1 {
		t.Error("Expected 1, Got:", c)
	}

	//WHEN
	signal.Unsubscribe(values)
	signal.Unsubscribe(empty)
	signal.Unsubscribe(errors)
	signal.Unsubscribe(completed)

	//THEN
	if v := len(signal.values); v != 0 {
		t.Error("Expected 0, Got:", v)
	}
	if e := len(signal.errors); e != 0 {
		t.Error("Expected 0, Got:", e)
	}
	if c := len(signal.completed); c != 0 {
		t.Error("Expected 0, Got:", c)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	wg.Add(2)

	//GIVEN
	sink := New()

	//WHEN/THEN
	signal := sink.Hold(666)

	signal.OnValue(func(ev Event) {
		t.Log(ev)
		if val := ev.(int); val != 1 {
			t.Error("Expected: 1, Got:", val)
		}
		wg.Done()
	})

	signal.OnComplete(func(ev Event) {
		t.Log(ev)
		if val := signal.Value(); val != 1 {
			t.Error("Expected: 1, Got:", val)
		}

		signal.Reset()

		if val := signal.Value(); val != 666 {
			t.Error("Expected: 666, Got:", val)
		}
		wg.Done()
	})

	sink.Send(1)
	time.Sleep(10 * time.Millisecond)
	sink.Send(Complete())

	wg.Wait()

}

func TestEmptyHandling(t *testing.T) {
	t.Parallel()

	//GIVEN
	sink := New()

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
	empty := Empty()
	sink.Send(empty)
	time.Sleep(10 * time.Millisecond)

	//THEN
	if val, ok := signal.Value().(EmptyEvent); ok && val != empty {
		t.Errorf("Expected: %#v, Got: %#v", empty, val)
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
	estimations := New()
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
