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

func OperatorFunc(do func(obs Observer, cur, last Event) Event, name string, init Event) Operator {
	return func(o Observable) Observable {
		return ObservableFunc(func(obs Observer) {
			input := make(chan Event)
			go func() {
				last := init
				DEBUG_CLEANUP("Starting " + name)
				for ev := range input {
					if IsCompleteEvent(ev) || IsErrorEvent(ev) {
						DEBUG_FLOW(name, ev)
						obs.Next(ev)
					} else {
						last = do(obs, ev, last)
						DEBUG_FLOW(name, last)
					}
				}
				DEBUG_CLEANUP("Closing " + name)
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

func Subscribe(onNext, onError, onComplete ObserverFunc) Operator {
	return func(o Observable) Observable {
		var input chan Event
		var last Event
		o.Observe(ObserverFunc(func(ev Event) {
			DEBUG_FLOW("Observed:", ev)
			switch ev.(type) {
			case CompleteEvent:
				onComplete(last)
				closeIfOpen(input)
			case ErrorEvent:
				onError(ev)
				closeIfOpen(input)
			default:
				last = ev
				onNext(ev)
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
