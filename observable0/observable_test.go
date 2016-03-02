package observable0

import (
	"flag"
	"testing"
	"time"
)

var CHATTY = func() (result bool) {
	flag.BoolVar(&result, "chatty", false, "For more debug output use -chatty")
	flag.Parse()
	return
}()

func debug(t *testing.T) {
	if !CHATTY {
		return
	}
	var seq int
	DEBUG_FLOW = func(prefix string, ev Event) {
		t.Logf("%s: %d -> %s, \t%v, \t%T", time.Now(), seq, prefix, ev, ev)
		seq++
	}
}

func print(t *testing.T, prefix string) func(Event) {
	return func(ev Event) {
		t.Logf("%s: %#v (%T)", prefix, ev, ev)
	}
}

type obj struct {
	v int
}

func (this obj) add(other obj) obj {
	return obj{this.v + other.v}
}

var testNils []Event = []Event{nil, nil, nil}
var testNumbers []Event = []Event{0, 1, 2, 3, 4, 5}
var testStrings []Event = []Event{"a", "b", "c", "d", "e", "f"}
var testObjects []Event = []Event{obj{0}, obj{1}, obj{2}}

func TestMap(t *testing.T) {
	debug(t)

	var tests = []struct {
		input []Event
		mapfn MapFunc
		check func(ev Event)
	}{
		{
			testNumbers,
			func(ev Event) Event { return ev.(int) * 2 },
			func(ev Event) {
				if n, ok := ev.(int); !(ok && n == 10) {
					t.Error("Expected: 10, Got:", n)
				}
			},
		},
		{
			testStrings,
			func(ev Event) Event { return ev.(string) + "oo" },
			func(ev Event) {
				if s, ok := ev.(string); !(ok && s == "foo") {
					t.Error("Expected: foo, Got:", s)
				}
			},
		},
		{
			testObjects,
			func(ev Event) Event { return obj{ev.(obj).v + 1000} },
			func(ev Event) {
				if o, ok := ev.(obj); !(ok && o.v == 1002) {
					t.Error("Expected: 1002, Got:", o.v)
				}
			},
		},
	}

	for _, test := range tests {
		o := Create(
			Just(test.input...),
			Map(test.mapfn),
		)
		SubscribeAndWait(o, nil, nil, func(ev Event) {
			test.check(ev)
		})
	}
}

func TestReduce(t *testing.T) {
	debug(t)

	var tests = []struct {
		input    []Event
		reducefn ReduceFunc
		init     Event
		check    func(ev Event)
	}{
		{
			testNumbers,
			func(ev1, ev2 Event) Event { return ev1.(int) + ev2.(int) },
			0,
			func(ev Event) {
				if n, ok := ev.(int); !(ok && n == 15) {
					t.Error("Expected: 15, Got:", n)
				}
			},
		},
		{
			testStrings,
			func(ev1, ev2 Event) Event { return ev1.(string) + ev2.(string) },
			"",
			func(ev Event) {
				if s, ok := ev.(string); !(ok && s == "abcdef") {
					t.Error("Expected: abcdef, Got:", s)
				}
			},
		},
		{
			testObjects,
			func(ev1, ev2 Event) Event { return ev1.(obj).add(ev2.(obj)) },
			obj{},
			func(ev Event) {
				if o, ok := ev.(obj); !(ok && o.v == 3) {
					t.Error("Expected: 6, Got:", o.v)
				}
			},
		},
	}

	for _, test := range tests {
		o := Create(
			Just(test.input...),
			Reduce(test.reducefn, test.init),
		)
		SubscribeAndWait(o, nil, nil, func(ev Event) {
			test.check(ev)
		})
	}
}

func TestAll(t *testing.T) {
	debug(t)
	o := Create(
		Just(0, 1, 2, 3, 4, 5, 6, 7, 8, 9),
		Map(func(ev Event) Event {
			return ev.(int) * ev.(int)
		}),
		Reduce(func(ev1, ev2 Event) Event {
			return ev1.(int) + ev2.(int)
		}, 0),
		Filter(func(ev Event) bool {
			return ev.(int) > 20
		}),
		Map(func(ev Event) Event {
			return ev.(int) + 10000
		}),
		// Debug(func(obs Observer, cur, last Event) {
		// 	t.Logf("DEBUG0 %#v, cur:%#v, last:%#v", obs, cur, last)
		// }),
	)

	SubscribeAndWait(o, print(t, "Next"), print(t, "Error"), print(t, "Completed"))
}
