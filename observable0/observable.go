package observable0

import (
	"sync"
)

var TRACE = func(prefix string, ev Event) {
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

// Observe method implements the Observable interface on the fly.
func (o ObservableFunc) Observe(obs Observer) {
	o(obs)
}

type Observer interface {
	Next(ev Event)
}

type ObserverFunc func(Event)

// Next method implements the Observer interface on the fly.
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
	}, "Map()", nil, true)
}

type ReduceFunc func(left, right Event) Event

func Reduce(reducefn ReduceFunc, init Event) Operator {
	return OperatorFunc(func(obs Observer, cur, last Event) Event {
		next := reducefn(last, cur)
		obs.Next(next)
		return next
	}, "Reduce()", init, true)
}

func Scan(reducefn ReduceFunc, init Event) Operator {
	return OperatorFunc(func(obs Observer, cur, last Event) Event {
		var next Event
		if IsCompleteEvent(cur) || IsErrorEvent(cur) {
			obs.Next(last)
			obs.Next(cur)

		} else {
			next = reducefn(last, cur)
		}
		return next
	}, "Scan()", init, false)
}

func Skip(n int) Operator {
	var count int
	return OperatorFunc(func(obs Observer, cur, last Event) Event {
		if count == n {
			obs.Next(cur)

		} else {
			count += 1
		}
		return cur
	}, "Skip()", nil, true)
}

func Take(n int) Operator {
	var count int
	return OperatorFunc(func(obs Observer, cur, last Event) Event {
		if count < n {
			obs.Next(cur)
			count += 1
		}
		return cur
	}, "Take()", nil, true)
}

type FilterFunc func(Event) bool

func Filter(filterfn FilterFunc) Operator {
	return OperatorFunc(func(obs Observer, cur, last Event) Event {
		if ok := filterfn(cur); ok {
			obs.Next(cur)
		}
		return cur
	}, "Filter()", nil, true)
}

type DebugFunc func(obs Observer, cur, last Event)

func Debug(debugfn DebugFunc) Operator {
	return OperatorFunc(func(obs Observer, cur, last Event) Event {
		debugfn(obs, cur, last)
		obs.Next(cur)
		return cur
	}, "DEBUG()", nil, true)
}

type BodyFunc func(obs Observer, cur, last Event) Event

func OperatorFunc(do BodyFunc, name string, init Event, shouldFilter bool) Operator {
	return func(o Observable) Observable {
		return ObservableFunc(func(obs Observer) {
			input := make(chan Event, 10)
			go func() {
				last := init
				TRACE("Starting", name)
				for ev := range input {
					if shouldFilter && (IsCompleteEvent(ev) || IsErrorEvent(ev)) {
						TRACE(name, ev)
						obs.Next(ev)
					} else {
						last = do(obs, ev, last)
						TRACE(name, last)
					}
				}
				TRACE("Closing", name)
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
		TRACE("Subscribe:", ev)
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
		TRACE("SubscribeAndWait:", ev)
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

func build(o Observable /*Func*/, ops ...Operator) Observable {
	var chained Observable = o

	for _, op := range ops {
		chained = op(chained)
	}
	return chained

}

func Create(o Observable /*Func*/, ops ...Operator) Observable {
	TRACE("Create", o)
	return build(o, ops...)
}

func Just(evts ...Event) ObservableFunc {
	return func(obs Observer) {
		for _, ev := range evts {
			TRACE("Just", ev)
			obs.Next(ev)
		}
		obs.Next(Complete())
	}
}
