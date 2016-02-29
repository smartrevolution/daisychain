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

var SAVED_DEBUG_FLOW func(string, Event)

func debug(t *testing.T) {
	if !CHATTY {
		return
	}
	var seq int
	SAVED_DEBUG_FLOW = DEBUG_FLOW
	DEBUG_FLOW = func(prefix string, ev Event) {

		t.Logf("%s: %d -> %s, \t%v, \t%T", time.Now(), seq, prefix, ev, ev)
		seq++
	}
}

func undebug() {
	DEBUG_FLOW = SAVED_DEBUG_FLOW
}

func print(t *testing.T, prefix string) func(Event) {
	return func(ev Event) {
		t.Logf("%s: %#v (%T)", prefix, ev, ev)
	}
}

func TestObservable(t *testing.T) {
	debug(t)
	defer undebug()

	o := Create(
		func(obs Observer) {
			for i := 0; i < 10; i++ {
				DEBUG_FLOW("Create", i)
				obs.Next(i)
			}
			obs.Next(Complete())
		},
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
		Debug(func(obs Observer, cur, last Event) {
			t.Logf("DEBUG0 %#v, cur:%#v, last:%#v", obs, cur, last)
		}),
	)

	SubscribeAndWait(o, print(t, "Next"), print(t, "Error"), print(t, "Completed"))
}
