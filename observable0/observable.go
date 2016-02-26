package observable0

var DEBUG_CLEANUP = func(msg string) {
	//do nothing, will be set while Testing
}

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

// EmptyEvent indicates an Empty event, like in "no value".
type EmptyEvent struct {
	Event
}

// Empty creates a new Event of type EmptyEvent.
func Empty() Event {
	return EmptyEvent{}
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

func Create(o ObservableFunc) Observable {
	DEBUG_FLOW("Create", o)
	return ObservableFunc(func(obs Observer) {
		o(obs)
	})
}

type Operator func(Observable) Observable

type MapFunc func(Event) Event

func Map(mapfn MapFunc) Operator {
	return func(o Observable) Observable {
		return ObservableFunc(func(obs Observer) {
			input := make(chan Event)
			go func() {
				DEBUG_CLEANUP("Starting Map()")
				for ev := range input {
					if IsCompleteEvent(ev) || IsErrorEvent(ev) {
						DEBUG_FLOW("Map:", ev)
						obs.Next(ev)
					} else {
						val := mapfn(ev)
						DEBUG_FLOW("Map:", val)
						obs.Next(val)
					}
				}
				DEBUG_CLEANUP("Closing Map()")
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

type ReduceFunc func(left, right Event) Event

func Reduce(reducefn ReduceFunc, init Event) Operator {
	return func(o Observable) Observable {
		return ObservableFunc(func(obs Observer) {
			input := make(chan Event)
			go func() {
				next := init
				DEBUG_CLEANUP("Starting Reduce()")
				for ev := range input {
					if IsCompleteEvent(ev) || IsErrorEvent(ev) {
						DEBUG_FLOW("Reduce:", ev)
						obs.Next(ev)
					} else {
						next = reducefn(next, ev)
						DEBUG_FLOW("Reduce:", next)
						obs.Next(next)
					}
				}
				DEBUG_CLEANUP("Closing Reduce()")
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

func Subscribe(onNext, onError, onComplete ObserverFunc) Operator {
	return func(o Observable) Observable {
		var input chan Event
		var last Event
		o.Observe(ObserverFunc(func(ev Event) {
			DEBUG_FLOW("Observed:", ev)
			if !IsCompleteEvent(ev) && !IsErrorEvent(ev) {
				last = ev
				onNext(ev)
			}
			if input != nil {
				input <- ev
				if IsCompleteEvent(ev) {
					onComplete(last)
					close(input)
				}
				if IsErrorEvent(ev) {
					onError(ev)
					close(input)
				}
			}
		}))
		return ObservableFunc(func(obs Observer) {
			input = make(chan Event)
			go func() {
				DEBUG_CLEANUP("Starting Subscribe()")
				for ev := range input {
					obs.Next(ev)
				}
				DEBUG_CLEANUP("Closing Subscribe()")
			}()
		})
	}
}

func consume() Observer {
	return ObserverFunc(func(ev Event) {
		DEBUG_FLOW("consume", ev)
	})
}

func Chain(o Observable, ops ...Operator) Observable {
	chained := o
	for _, op := range ops {
		chained = op(chained)
	}
	chained.Observe(consume())
	return chained
}

func main() {

}
