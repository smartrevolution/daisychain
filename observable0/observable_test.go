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

		t.Logf("%s: %d -> %s, \t\t%v, \t\t%T", time.Now(), seq, prefix, ev, ev)
		seq++
	}

	DEBUG_CLEANUP = func(msg string) {
		t.Log(msg)
	}
}

func print(t *testing.T, prefix string) func(Event) {
	return func(ev Event) {
		t.Logf("%s: %#v (%T)", prefix, ev, ev)
	}
}

func TestObservable(t *testing.T) {
	debug(t)
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
		Subscribe(print(t, "next"), print(t, "error"), print(t, "completed")),
	)
	time.Sleep(time.Second)
}
