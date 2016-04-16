package daisychain

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
	TRACE = func(prefix string, ev Event) {
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

func TestAnon(t *testing.T) {
	var item struct {
		foo string
		bar string
	}
	o := Create(
		ObservableFunc(func(obs Observer) {
			item.foo = "foo"
			item.bar = "bar"
			obs.Next(item)
			obs.Next(Complete())
		}),
	)
	SubscribeAndWait(o, nil, nil, func(ev Event) {
		if s, ok := ev.(struct {
			foo string
			bar string
		}); !ok {
			t.Errorf("Expected: %#v, Got: %#v (%T)", ev, s, s)
		}
	})
}

func TestMap(t *testing.T) {
	debug(t)

	var tests = []struct {
		input []Event
		mapfn MapFunc
		check func(ev Event)
	}{
		{
			testNils,
			func(ev Event) Event {
				if _, ok := ev.(int); !ok {
					return nil
				}
				return 999
			},
			func(ev Event) {
				if ev != nil {
					t.Error("Expected: nil, Got:", ev)
				}
			},
		},
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

func TestScan(t *testing.T) {
	debug(t)

	var tests = []struct {
		input    []Event
		reducefn ReduceFunc
		init     Event
		check    func(ev Event)
	}{
		{
			testNils,
			func(ev1, ev2 Event) Event {
				if _, ok := ev1.(int); !ok {
					return nil
				}
				return 999
			},
			nil,
			func(ev Event) {
				if ev != nil {
					t.Error("Expected: nil, Got:", ev)
				}
			},
		},
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
			Scan(test.reducefn, test.init),
		)
		SubscribeAndWait(o, nil, nil, func(ev Event) {
			test.check(ev)
		})
	}
}

func TestFlatMap(t *testing.T) {
	debug(t)
	o := Create(
		Just(1, 2, 3, 4, 5),
		FlatMap(func(ev Event) Observable {
			return Create(
				Just(ev, ev),
			)
		}),
		Reduce(func(ev1, ev2 Event) Event {
			return ev1.(int) + ev2.(int)
		}, 0),
	)
	SubscribeAndWait(o, print(t, "Next"), print(t, "Error"), func(ev Event) {
		if n, ok := ev.(int); !(ok && n == 30) {
			t.Error("Expected: 30, Got:", n)
		}
	})
}

func TestSkip(t *testing.T) {
	debug(t)
	o := Create(
		Just(1, 2, 3, 4, 5),
		Skip(3),
		Reduce(func(ev1, ev2 Event) Event {
			return ev1.(int) + ev2.(int)
		}, 0),
	)
	SubscribeAndWait(o, print(t, "Next"), print(t, "Error"), func(ev Event) {
		if n, ok := ev.(int); !(ok && n == 9) {
			t.Error("Expected: 9, Got:", n)
		}
	})
}

func TestTake(t *testing.T) {
	debug(t)
	o := Create(
		Just(1, 2, 3, 4, 5),
		Take(3),
		Reduce(func(ev1, ev2 Event) Event {
			return ev1.(int) + ev2.(int)
		}, 0),
	)
	SubscribeAndWait(o, print(t, "Next"), print(t, "Error"), func(ev Event) {
		if n, ok := ev.(int); !(ok && n == 6) {
			t.Error("Expected: 6, Got:", n)
		}
	})
}

func TestToVector(t *testing.T) {
	debug(t)
	o := Create(
		Just(1, 2, 3, 4, 5),
		ToVector(),
	)
	SubscribeAndWait(o, print(t, "Next"), print(t, "Error"), func(ev Event) {
		if n, ok := ev.([]Event); !(ok && len(n) == 5) {
			t.Error("Expected: 5, Got:", n)
		}
	})
}

func TestToMap(t *testing.T) {
	debug(t)
	o := Create(
		Just(1, 2, 3, 4, 5),
		ToMap(func(ev Event) string {
			if ev.(int)%2 == 0 {
				return "even"
			}
			return "odd"
		}),
	)
	SubscribeAndWait(o, print(t, "Next"), print(t, "Error"), func(ev Event) {
		if n, ok := ev.(map[string][]Event); !(ok && len(n) == 2) {
			t.Error("Expected: 2, Got:", n)
		}
	})
}

func TestDistinctAndCount(t *testing.T) {
	debug(t)
	o := Create(
		Just("a", "b", "a", "b", "c"),
		Distinct(func(ev Event) string {
			return ev.(string)
		}),
		Count(),
	)

	SubscribeAndWait(o, print(t, "Next"), print(t, "Error"), func(ev Event) {
		if n, ok := ev.(int64); !(ok && n == 3) {
			t.Error("Expected: 3, Got:", n)
		}
	})
}

func TestZip(t *testing.T) {
	debug(t)
	o1 := Create(
		Just(1, 2, 3),
		Map(func(ev Event) Event {
			return ev
		}),
	)
	o2 := Create(
		Just(4, 5, 6),
		Map(func(ev Event) Event {
			return ev
		}),
	)

	o := Create(
		Just(10, 20, 30),
		Zip(func(evs ...Event) Event {
			var sum int
			for _, ev := range evs {
				sum += ev.(int)
			}
			return sum
		}, o1, o2),
		Scan(func(ev1, ev2 Event) Event {
			return ev1.(int) + ev2.(int)
		}, 0),
	)
	expected := [3]int{15, 42, 81}
	var got [3]int
	var i int
	SubscribeAndWait(o, func(ev Event) {
		got[i] = ev.(int)
		i += 1
	}, nil, nil)

	if expected != got {
		t.Errorf("Expected: %v, Got: %v", expected, got)
	}
}

func TestAll(t *testing.T) {
	debug(t)
	o := Create(
		Just(0, 1, 2, 3, 4, 5, 6, 7, 8, 9),
		Map(func(ev Event) Event {
			return ev.(int) * ev.(int)
		}),
		Scan(func(ev1, ev2 Event) Event {
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

// func TestCleanup(t *testing.T) {
// 	var cnt
// 	o := Create(
// 		ObservableFunc(func(obs Observer) {

// 		}),
// 	)
// }

func TestCreate(t *testing.T) {
	debug(t)
	o := Create(
		ObservableFunc(func(obs Observer) {
			for _, n := range []int{10, 20, 30} {
				obs.Next(n)
			}
			obs.Next(Complete())
		}),
	)

	SubscribeAndWait(o, print(t, "Next"), print(t, "Error"), func(ev Event) {
		if n, ok := ev.(int); !(ok && n == 30) {
			t.Error("Expected: 30, Got:", n)
		}
	})
}

func TestSubscribe(t *testing.T) {
	debug(t)
	o := Create(
		Just(11, 22, 33),
	)
	Subscribe(o, print(t, "Next"), print(t, "Error"), func(ev Event) {
		if n, ok := ev.(int); !(ok && n == 33) {
			t.Error("Expected: 33, Got:", n)
		}
	})
	SubscribeAndWait(o, print(t, "Next"), print(t, "Error"), func(ev Event) {
		if n, ok := ev.(int); !(ok && n == 33) {
			t.Error("Expected: 33, Got:", n)
		}
	})
}

func TestJust(t *testing.T) {
	debug(t)
	o := Create(
		Just(1, 2, 3),
	)
	SubscribeAndWait(o, func(ev Event) {
		t.Log(ev)
	}, nil, nil)
}

func TestComposition(t *testing.T) {
	debug(t)
	o1 := Create(
		Just(2, 3, 5, 7, 11),
		Map(func(ev Event) Event {
			return ev.(int) + 1
		}),
	)
	o2 := Create(
		o1,
		Scan(func(ev1, ev2 Event) Event {
			return ev1.(int) + ev2.(int)
		}, 0),
	)

	SubscribeAndWait(o2, print(t, "Next"), print(t, "Error"), func(ev Event) {
		if n, ok := ev.(int); !(ok && n == 33) {
			t.Error("Expected: 33, Got:", n)
		}
	})
}
func TestReduce(t *testing.T) {
	debug(t)
	o := Create(
		Just(1, 2, 3),
		Reduce(func(ev1, ev2 Event) Event {
			return ev1.(int) + ev2.(int)
		}, 0),
	)
	SubscribeAndWait(o, print(t, "Next"), print(t, "Error"), func(ev Event) {
		if n, ok := ev.(int); !(ok && n == 6) {
			t.Error("Expected: 6, Got:", n)
		}
	})
}

func addInts(upto int) Observable {
	return Create(
		ObservableFunc(func(obs Observer) {
			for i := 0; i < upto; i++ {
				obs.Next(i)
			}
			obs.Next(Complete())
		}),
		Scan(func(ev1, ev2 Event) Event {
			return ev1.(int) + ev2.(int)
		}, 0),
	)
}

func benchmarkAdd(i int, b *testing.B) {
	o := addInts(i)
	for n := 0; n < b.N; n++ {
		SubscribeAndWait(o, nil, nil, nil)
	}
}

func BenchmarkAdd10(b *testing.B)      { benchmarkAdd(10, b) }
func BenchmarkAdd1000(b *testing.B)    { benchmarkAdd(1000, b) }
func BenchmarkAdd1000000(b *testing.B) { benchmarkAdd(1000000, b) }
