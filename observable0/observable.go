package observable0

import (
	"sync"
)

var DEBUG_FLOW = func(prefix string, ev Event) {
	//do nothing, will be set while testing
}

type Event interface{}

// CompleteEvent indicates that no more Events will be send.
type CompleteEvent struct {
	Event
}

// Complete creates a new Event of type CompleteEvent.
func Complete() Event {
	return CompleteEvent{}
}

// IsCompleteEvent returns true if ev is a CompleteEvent.
func IsCompleteEvent(ev Event) bool {
	_, isCompleteEvent := ev.(CompleteEvent)
	return isCompleteEvent
}

// IsErrorEvent returns true if ev is a CompleteEvent.
func IsErrorEvent(ev Event) bool {
	_, isErrorEvent := ev.(ErrorEvent)
	return isErrorEvent
}

// ErrorEvent indicates an Error.
type ErrorEvent struct {
	Event
	msg string
}

// Error creates a new Event of type ErrorEvent.
func Error(msg string) Event {
	return ErrorEvent{msg: msg}
}

type Observable interface {
	Observe(Observer)
}

type ObservableFunc func(Observer)

func (o ObservableFunc) Observe(obs Observer) {
	o(obs)
}

type Observer interface {
	Next(ev Event)
}

type ObserverFunc func(Event)

func (o ObserverFunc) Next(ev Event) {
	o(ev)
}

type Operator func(Observable) Observable

type MapFunc func(Event) Event

func Map(mapfn MapFunc) Operator {
	return OperatorFunc(func(obs Observer, cur, last Event) Event {
		next := mapfn(cur)
		obs.Next(next)
		return next
	}, "Map()", nil)
}

type ReduceFunc func(left, right Event) Event

func Reduce(reducefn ReduceFunc, init Event) Operator {
	return OperatorFunc(func(obs Observer, cur, last Event) Event {
		next := reducefn(last, cur)
		obs.Next(next)
		return next
	}, "Reduce()", init)
}

type FilterFunc func(Event) bool

func Filter(filterfn FilterFunc) Operator {
	return OperatorFunc(func(obs Observer, cur, last Event) Event {
		if ok := filterfn(cur); ok {
			obs.Next(cur)
		}
		return cur
	}, "Filter()", nil)
}

type DebugFunc func(obs Observer, cur, last Event)

func Debug(debugfn DebugFunc) Operator {
	return OperatorFunc(func(obs Observer, cur, last Event) Event {
		debugfn(obs, cur, last)
		obs.Next(cur)
		return cur
	}, "DEBUG()", nil)
}

func OperatorFunc(do func(obs Observer, cur, last Event) Event, name string, init Event) Operator {
	return func(o Observable) Observable {
		return ObservableFunc(func(obs Observer) {
			input := make(chan Event)
			go func() {
				last := init
				DEBUG_FLOW("Starting", name)
				for ev := range input {
					if IsCompleteEvent(ev) || IsErrorEvent(ev) {
						DEBUG_FLOW(name, ev)
						obs.Next(ev)
					} else {
						last = do(obs, ev, last)
						DEBUG_FLOW(name, last)
					}
				}
				DEBUG_FLOW("Closing", name)
			}()
			o.Observe(ObserverFunc(func(ev Event) {
				input <- ev
				if IsCompleteEvent(ev) || IsErrorEvent(ev) {
					close(input)
				}
			}))
		})
	}
}

func closeIfOpen(input chan Event) {
	if input != nil {
		close(input)
	}
}

func callIfNotNil(onEvent ObserverFunc, ev Event) {
	if onEvent != nil {
		onEvent(ev)
	}
}

func Subscribe(o Observable, onNext, onError, onComplete ObserverFunc) {
	var last Event
	o.Observe(ObserverFunc(func(ev Event) {
		DEBUG_FLOW("Subscribe:", ev)
		switch ev.(type) {
		case CompleteEvent:
			callIfNotNil(onComplete, last)
		case ErrorEvent:
			callIfNotNil(onError, ev)
		default:
			last = ev
			callIfNotNil(onNext, ev)
		}
	}))
}

func SubscribeAndWait(o Observable, onNext, onError, onComplete ObserverFunc) {
	var wg sync.WaitGroup
	var last Event

	wg.Add(1)
	o.Observe(ObserverFunc(func(ev Event) {
		DEBUG_FLOW("SubscribeAndWait:", ev)
		switch ev.(type) {
		case CompleteEvent:
			callIfNotNil(onComplete, last)
			wg.Done()
		case ErrorEvent:
			callIfNotNil(onError, ev)
		default:
			last = ev
			callIfNotNil(onNext, ev)
		}
	}))
	wg.Wait()
}

func build(o ObservableFunc, ops ...Operator) Observable {
	var chained Observable
	chained = ObservableFunc(func(obs Observer) {
		o(obs)
	})
	for _, op := range ops {
		chained = op(chained)
	}
	return chained

}

func Create(o ObservableFunc, ops ...Operator) Observable {
	DEBUG_FLOW("Create", o)
	return ObservableFunc(func(obs Observer) {
		chained := build(o, ops...)
		chained.Observe(obs)
	})
}

func Just(evts ...Event) ObservableFunc {
	return func(obs Observer) {
		for _, ev := range evts {
			DEBUG_FLOW("From", ev)
			obs.Next(ev)
		}
		obs.Next(Complete())
	}
}
