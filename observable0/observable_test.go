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
var SAVED_DEBUG_CLEANUP func(string)

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
	SAVED_DEBUG_CLEANUP = DEBUG_CLEANUP
	DEBUG_CLEANUP = func(msg string) {
		t.Log(msg)
	}
}

func undebug() {
	DEBUG_FLOW = SAVED_DEBUG_FLOW
	DEBUG_CLEANUP = SAVED_DEBUG_CLEANUP
}

func print(t *testing.T, prefix string) func(Event) {
	return func(ev Event) {
		t.Logf("%s: %#v (%T)", prefix, ev, ev)
	}
}

func TestObservable(t *testing.T) {
	debug(t)
	defer undebug()
	Chain(
		Create(func(obs Observer) {
			for i := 0; i < 10; i++ {
				DEBUG_FLOW("Create", i)
				obs.Next(i)
			}
			obs.Next(Complete())
		}),
		Map(func(ev Event) Event {
			return ev.(int) * ev.(int)
		}),
		Reduce(func(ev1, ev2 Event) Event {
			return ev1.(int) + ev2.(int)
		}, 0),
		Subscribe(print(t, "Next"), print(t, "Error"), print(t, "Completed")),
		Map(func(ev Event) Event {
			return ev
		}),
	)
	time.Sleep(time.Second)
}
