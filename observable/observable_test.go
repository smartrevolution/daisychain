package observable

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestClose(t *testing.T) {
	//t.Parallel()

	//Close() O
	observable := New()
	mapped := observable.Map(func(ev Event) Event {
		return ev //do nothing
	})

	observable.From(1, 2, 3, 4, 5)
	observable.Close()

	observable = New()
	mapped = observable.Map(func(ev Event) Event {
		return ev //do nothing
	})
	signal := mapped.Hold(0)

	observable.From(1, 2, 3, 4, 5)
	signal.Close()
	//not a really intelligent test...but it doesn't crash
	//and that is a good sign. Actually I tested this with more
	//debug output to see the inner workings running well.
	//I need to create a better test for that.
}

func TestSubscribers(t *testing.T) {
	//t.Parallel()
	//GIVEN
	parent := newO()
	child := newO()

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
	//t.Parallel()

	//GIVEN
	observable := New()
	signal := observable.Hold(0)

	//WHEN
	observable.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	if val := signal.Value(); val != 9 {
		t.Error("Expected: 9, Got:", val)
	}
}

func testNewSub(t *testing.T, wg *sync.WaitGroup) *O {
	o := New()
	o.Subscribe(nil, nil, func(ev Event) {
		DEBUG_FLOW("Subscribe", ev)
		if val, ok := ev.(int); ok && val != 9 {
			t.Errorf("Expected: 9, Got: %d", val)
		} else {
			t.Errorf("Expected: int (9), Got: %T (%v)", ev, ev)
		}
	})
	o.Subscribe(nil, nil, func(ev Event) {
		wg.Done()
	})
	return o
}

func testNewMapReduceSub(t *testing.T, wg *sync.WaitGroup) *O {
	o := New()
	m := o.Map(func(ev Event) Event {
		return ev.(int) * ev.(int)
	})
	r := m.Reduce(func(e1, e2 Event) Event {
		return e1.(int) + e2.(int)
	}, 0)
	r.Subscribe(nil, nil, func(ev Event) {
		DEBUG_FLOW("Subscribe", ev)
		if val, ok := ev.(int); ok && val != 9 {
			t.Errorf("Expected: 9, Got: %d", val)
		} else {
			t.Errorf("Expected: int (9), Got: %T (%v)", ev, ev)
		}
	})
	o.Subscribe(nil, nil, func(ev Event) {
		wg.Done()
	})
	return o
}

var DEBUG = true

//var DEBUG = false

func debug(t *testing.T) {
	if !DEBUG {
		return
	}
	var seq int
	DEBUG_FLOW = func(prefix string, ev Event) {

		t.Logf("%s: %d -> %s, \t\t%v, \t\t%T", time.Now(), seq, prefix, ev, ev)
		seq++
	}
}

func TestNewSubSend(t *testing.T) {
	debug(t)
	var wg sync.WaitGroup
	o := testNewSub(t, &wg)
	wg.Add(1)
	for i := 0; i < 10; i++ {
		o.Send(i)
	}
	o.Send(Complete())
	wg.Wait()
}

func TestNewSubFrom(t *testing.T) {
	debug(t)
	var wg sync.WaitGroup
	o := testNewSub(t, &wg)
	wg.Add(1)
	o.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	o.Send(Complete())
	wg.Wait()
}

func TestNewSubJust(t *testing.T) {
	debug(t)
	var wg sync.WaitGroup
	o := testNewSub(t, &wg)
	wg.Add(1)
	o.Just(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	wg.Wait()
}

func TestNewMapReduceSubJust(t *testing.T) {
	debug(t)
	var wg sync.WaitGroup
	o := testNewMapReduceSub(t, &wg)
	wg.Add(1)
	o.Just(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	wg.Wait()
}

func TestMap(t *testing.T) {
	//t.Parallel()

	//GIVEN
	observable := New()

	squared := observable.Map(func(ev Event) Event {
		return ev.(int) * ev.(int)
	})

	signal := squared.Hold(0)

	//WHEN
	observable.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	if val := signal.Value(); val != 81 {
		t.Error("Expected: 81, Got:", val)
	}
}

func TestFlatMap(t *testing.T) {
	observable := New()
	mapped := observable.Map(func(ev Event) Event {
		return ev.(int) * ev.(int)
	})
	flatmapped := mapped.FlatMap(func(ev Event) *O {
		observable = New().
			Map(func(ev Event) Event {
			return ev.(int) + 1
		})
		return observable
	})
	flatmapped.Subscribe(nil, nil, func(ev Event) {
		if v, ok := ev.(int); !ok && v != 2 && v != 5 && v != 10 {
			t.Error("Expected 2, 5 or 10: Got:", v)
		}
	}).Just(1, 2, 3)

}

func TestReduce(t *testing.T) {
	//t.Parallel()

	//GIVEN
	observable := New()

	squared := observable.Reduce(func(a, b Event) Event {
		return a.(int) + b.(int)
	}, 100)

	signal := squared.Hold(0)

	//WHEN
	observable.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	if val := signal.Value(); val != 145 {
		t.Error("Expected: 145, Got:", val)
	}
}

func TestFilter(t *testing.T) {
	//t.Parallel()

	//GIVEN
	observable := New()

	evenNums := observable.Filter(func(ev Event) bool {
		return ev.(int)%2 == 0
	})

	signal := evenNums.Hold(0)

	//WHEN
	observable.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	if val := signal.Value(); val != 8 {
		t.Error("Expected: 8, Got:", val)
	}
}

func TestCompleteBehavior(t *testing.T) {
	//t.Parallel()

	//GIVEN
	s0 := New()

	var wg sync.WaitGroup

	//WHEN/THEN
	//map should not see complete events
	s1 := s0.Map(func(ev Event) Event {
		switch ev.(type) {
		case CompleteEvent:
			t.Error("Map: Did not expect this event:", ev)
		default:
			t.Log("Map", ev)
		}
		return ev
	})

	//reduce should not see complete events
	s2 := s1.Reduce(func(left, right Event) Event {
		switch right.(type) {
		case CompleteEvent:
			t.Error("Reduce: Did not expect this event:", right)
		default:
			t.Log("Reduce", right)
		}
		return right
	}, 0)

	//filter "sees" complete events
	s3 := s2.Filter(func(ev Event) bool {
		switch ev.(type) {
		case CompleteEvent:
			wg.Done() // filter "sees" complete events
		default:
			t.Log("Filter", ev)
		}
		return true
	})

	wg.Add(2)
	signal := s3.Hold(0)
	signal.OnComplete(func(ev Event) {
		wg.Done()
	})

	s0.From(0, 1, 2)
	s0.Send(Complete())

	wg.Wait()
}

func TestThrottle(t *testing.T) {
	//t.Parallel()

	//GIVEN
	observable := New()
	mapped := observable.Map(func(ev Event) Event {
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
	observable.Send(1)
	observable.Send(1)
	observable.Send(1)
	time.Sleep(15 * time.Millisecond)
}

func TestMerge(t *testing.T) {
	//t.Parallel()

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
	//t.Parallel()

	//GIVEN
	observable := New()
	collected := observable.Collect()
	signal := collected.Hold(Empty())

	//WHEN
	observable.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	expected := []Event{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if val, ok := signal.Value().([]Event); len(val) != len(expected) || !ok {
		t.Errorf("Expected: %#v, Got: %#v", expected, val)
	}
}

func TestGroupBy(t *testing.T) {
	//t.Parallel()

	//GIVEN
	observable := New()
	grouped := observable.GroupBy(func(ev Event) string {
		if ev.(int)%2 == 0 {
			return "even"
		}
		return "odd"
	})
	signal := grouped.Hold(Empty())

	//WHEN
	observable.From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	time.Sleep(5 * time.Millisecond)

	//THEN
	if evenOdd, ok := signal.Value().(map[string][]Event); len(evenOdd) != 2 || !ok {
		t.Log(signal.Value())
		t.Errorf("Expected: map[even:[0 2 4 6 8] odd:[1 3 5 7 9]], Got: %#v", evenOdd)
	}
}

func TestDistinct(t *testing.T) {
	//	//t.Parallel()

	New().
		Distinct(func(ev Event) string {
		if ev.(int)%2 == 0 {
			return "even"
		}
		return "odd"
	}).
		Subscribe(nil, nil, func(ev Event) {
		if evenOdd, ok := ev.(map[string]Event); !ok && len(evenOdd) != 2 {
			t.Log(evenOdd)
			t.Errorf("Expected: map[even:8 odd:9], Got: %#v", evenOdd)
		}
	}).
		From(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)

}

func xTestSignalOnChange(t *testing.T) {
	//t.Parallel()
	var wg sync.WaitGroup

	//GIVEN
	observable := New()

	//WHEN/THEN
	signal := observable.Hold(0)
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
	observable.Send(1)
	observable.Send(Empty())
	observable.Send(Error("Testerror"))
	observable.Send(Complete())
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
	//t.Parallel()

	var wg sync.WaitGroup
	wg.Add(2)

	//GIVEN
	observable := New()

	//WHEN/THEN
	signal := observable.Hold(666)

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

	observable.Send(1)
	time.Sleep(10 * time.Millisecond)
	observable.Send(Complete())

	wg.Wait()

}

func TestEmptyHandling(t *testing.T) {
	//t.Parallel()

	//GIVEN
	observable := New()

	//WHEN
	signal := observable.Hold(666)

	//THEN
	if val := signal.Value(); val != 666 {
		t.Error("Expected: 666, Got:", val)
	}

	//WHEN
	observable.Send(1)
	time.Sleep(10 * time.Millisecond)

	//THEN
	if val := signal.Value(); val != 1 {
		t.Error("Expected: 1, Got:", val)
	}

	//WHEN
	empty := Empty()
	observable.Send(empty)
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
	//t.Parallel()

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

func TestHoldAsO(t *testing.T) {
	New().
		Map(func(ev Event) Event {
		return ev.(int) * ev.(int)
	}).
		Hold(0).
		Observable().
		Reduce(func(e1, e2 Event) Event {
		return e1.(int) + e2.(int)
	}, 0).
		Subscribe(func(ev Event) {
		if val, _ := ev.(int); val != 1 && val != 5 && val != 14 {
			t.Error(val)
		}
	}, nil, nil).
		From(1, 2, 3)

	time.Sleep(50 * time.Millisecond)
}

func TestHoldCollectAsO(t *testing.T) {
	observable := New()
	mapped := observable.Map(func(ev Event) Event {
		return ev.(int) * ev.(int)
	})
	collected := mapped.Collect()

	collected.Subscribe(func(ev Event) {
		t.Log(ev)
	}, nil, nil)

	signal := collected.Hold(Empty())
	observable.From(1, 2, 3)

	time.Sleep(10 * time.Millisecond)
	t.Log(signal.Value())
}

func TestNoSignal(t *testing.T) {
	New().
		Map(func(ev Event) Event {
		return ev.(int) * ev.(int)
	}).
		Subscribe(func(ev Event) {
		if val, _ := ev.(int); val != 1 && val != 4 && val != 9 {
			t.Error(val)
		}
	}, nil, nil).
		Reduce(func(e1, e2 Event) Event {
		return e1.(int) + e2.(int)
	}, 0).
		Subscribe(func(ev Event) {
		if val, _ := ev.(int); val != 1 && val != 5 && val != 14 {
			t.Error(val)
		}
	}, nil, nil).
		From(1, 2, 3)

	time.Sleep(50 * time.Millisecond)
}

func TestConnect(t *testing.T) {
	o := Create(func(o *O) {
		o.Just(1, 2, 3, 4, 5)
	})

	mapped := o.Map(func(ev Event) Event {
		return ev.(int) * ev.(int)
	})

	var wg sync.WaitGroup

	mapped.Subscribe(nil, nil, func(ev Event) {
		if val, _ := ev.(int); val != 25 {
			t.Error("Expected 25, Got:", val)
		}
		wg.Done()
	})

	wg.Add(1)
	mapped.Connect()
	wg.Wait()
	wg.Add(1)
	mapped.Connect()
	wg.Wait()
}

// func TestCacheReplay(t *testing.T) {
// 	o := Create(func(o *O) {
// 		o.Just(1, 2, 3, 4, 5)
// 	})

// 	mapped := o.Map(func(ev Event) Event {
// 		return ev.(int) * ev.(int)
// 	})

// 	cached := mapped.Cache()

// 	var wg sync.WaitGroup

// 	cached.Subscribe(nil, nil, func(ev Event) {
// 		if val, _ := ev.(int); val != 25 {
// 			t.Error("Expected 25, Got:", val)
// 		}
// 		wg.Done()
// 	})

// 	wg.Add(1)
// 	mapped.Connect()
// 	wg.Wait()
// 	wg.Add(1)
// 	mapped.Replay()
// 	wg.Wait()
// 	// wg.Add(1)
// 	// mapped.Refresh()
// 	// wg.Wait()
// }
