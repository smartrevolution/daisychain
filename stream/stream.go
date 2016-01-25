package stream

import (
	"log"
	//	"runtime"
	"sync"
	"time"
)

type Event interface{}

type subscribers map[*Stream]interface{}

func (s *Stream) subscribe(child *Stream) {
	s.Lock()
	defer s.Unlock()
	child.root = s.root
	s.subs[child] = struct{}{}
}

func (s *Stream) unsubscribe(child *Stream) {
	s.Lock()
	defer s.Unlock()
	delete(s.subs, child)
}

func (s *Stream) publish(ev Event) {
	s.RLock()
	defer s.RUnlock()
	for child, _ := range s.subs {
		child.send(ev)
	}
}

// CompleteEvent indicates that no more Events will be send.
type CompleteEvent struct {
	Event
}

// Complete creates a new Event of type CompleteEvent.
func Complete() Event {
	return CompleteEvent{}
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

// Observable
type Stream struct {
	sync.RWMutex
	in   chan Event
	subs subscribers
	root *Stream
}

// Close stops all running computations and cleans up.
// If you don't call this, bad things will happen,
// because of all the go routines that keep on running...
func (s *Stream) Close() {
	s.root.close()
}

// newStream is a constructor for streams.
func newStream() *Stream {
	return &Stream{
		in:   make(chan Event),
		subs: make(subscribers),
	}
}

// If the runtime discards a Observable, all depending streams are discarded , too.
// func sinkFinalizer(s *Observable) {
// 	log.Println("DEBUG: Called Finalizer")
// 	s.close()
// }

// New generates a new Observable.
func New() *Stream {
	s := newStream()
	s.root = s
	//runtime.SetFinalizer(s, sinkFinalizer)
	go func() {
		for ev := range s.in {
			s.publish(ev)
		}
		log.Println("DEBUG: Closing Observable")
	}()

	return s
}

// Send sends an Event down the stream
func (s *Stream) Send(ev Event) {
	s.root.in <- ev
}

// Complete sends a CompleteEvent down the stream
func (s *Stream) Complete() {
	s.in <- Complete()
}

// Error sends an ErrorEvent down the stream
func (s *Stream) Error(msg string) {
	s.in <- Error(msg)
}

// Empty sends an EmptyEvent down the stream
func (s *Stream) Empty() {
	s.in <- Empty()
}

// send sends an Event down the stream.
func (s *Stream) send(ev Event) {
	s.in <- ev
}

// close will unsubscribe all children recursively.
func (s *Stream) close() {
	for child := range s.subs {
		s.unsubscribe(child)
		child.close()
		close(child.in)
	}
}

type FeederFunc func(s *Stream)

// Connect starts FeederFunc as a go routine.
// Use Send(), Empty(), Error(), Complete() inside
// your FeederFunc to send events async down the stream.
func (s *Stream) Connect(feeder FeederFunc) *Stream {
	go feeder(s.root)
	return s
}

// From sends evts down the stream and blocks until
// all events are sent.
func (s *Stream) From(evts ...Event) {
	for _, ev := range evts {
		s.root.Send(ev)
	}
}

// Just sends evts down the stream and blocks until
// all events are sent, then it sends a final CompleteEvent.
func (s *Stream) Just(evts ...Event) {
	s.From(evts...)
	s.root.Complete()
}

// isCompleteEvent returns true if ev is a CompleteEvent.
func isCompleteEvent(ev Event) bool {
	_, isCompleteEvent := ev.(CompleteEvent)
	return isCompleteEvent
}

// MapFunc is the function signature used by Map.
type MapFunc func(Event) Event

// Map returns an Observable that applies MapFunc to each event emitted by the source Observable and emits the results of these function applications.
func (s *Stream) Map(mapfn MapFunc) *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
		for ev := range res.in {
			if ev != nil {
				if !isCompleteEvent(ev) {
					val := mapfn(ev)
					res.publish(val)
				} else {
					res.publish(ev)
				}
			}
		}
		log.Println("DEBUG: Closing Map()")
	}()

	return res
}

// FlatMapFunc is the function signature used by FlatMap.
type FlatMapFunc func(ev Event) *Stream

// FlatMap returns an Observable that emits events based on applying FlatMapFunc to each event emitted by the source Observable, where that function returns an Observable, and then emitting the results of this Observable.
func (s *Stream) FlatMap(flatmapfn FlatMapFunc) *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
		for ev := range res.in {
			if ev != nil {
				o := flatmapfn(ev)
				onValue := func(o *Stream) func(ev Event) {
					return func(ev Event) {
						res.publish(ev)
					}
				}
				onError := func(o *Stream) func(ev Event) {
					return func(ev Event) {
						res.publish(ev)
						o.Close()
					}
				}
				onComplete := func(o *Stream) func(ev Event) {
					return func(ev Event) {
						res.publish(ev)
						o.Close()
					}
				}
				o.Subscribe(onValue(o), onError(o), onComplete(o))
				o.Just(ev)
			}
		}
		log.Println("DEBUG: Closing FlatMap()")
	}()

	return res
}

//ReduceFunc is the function signature used by Reduce().
type ReduceFunc func(left, right Event) Event

// Reduce returns an Observable that applies ReduceFunc to the init Event and first event emitted by a source Observable, then feeds the result of that function along with the second event emitted by the source Observable into the same function, and emits each result.
func (s *Stream) Reduce(reducefn ReduceFunc, init Event) *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
		val := init
		for ev := range res.in {
			if ev != nil {
				if !isCompleteEvent(ev) {
					val = reducefn(val, ev)
					res.publish(val)
				} else {
					res.publish(ev)
				}
			}
		}
		log.Println("DEBUG: Closing Reduce()")
	}()

	return res
}

// FilterFunc is the function signature used by Filter().
type FilterFunc func(Event) bool

// Filter returns events emitted by an Observable by only emitting those that satisfy the specified predicate FilterFunc.
func (s *Stream) Filter(filterfn FilterFunc) *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
		for ev := range res.in {
			if (ev != nil && filterfn(ev)) || isCompleteEvent(ev) {
				res.publish(ev)
			}
		}
		log.Println("DEBUG: Closing Filter()")
	}()

	return res
}

type LookupFunc func(ev Event, signal *Signal) Event

// Lookup
func (s *Stream) Lookup(lookupfn LookupFunc, signal *Signal) *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
		for ev := range res.in {
			if ev != nil {
				if !isCompleteEvent(ev) {
					val := lookupfn(ev, signal)
					res.publish(val)
				} else {
					res.publish(ev)
				}
			}
		}
		log.Println("DEBUG: Closing Lookup()")
	}()

	return res
}

// Merge merges two source observables into one observable.
// Merge emits an event each time either of
// the source observables emits an event.
func (s *Stream) Merge(other *Stream) *Stream {
	res := newStream()
	s.subscribe(res)
	other.subscribe(res)

	go func() {
		for ev := range res.in {
			if ev != nil {
				res.publish(ev)
			}
		}
		log.Println("DEBUG: Closing Merge()")
	}()

	return res
}

// Throttle buffers events and returns them every d time durations.
// The emitted event is []Event.
// The result can be empty, when no events were received in the last interval.
func (s *Stream) Throttle(d time.Duration) *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
		ticker := time.NewTicker(d)
		var events []Event
		for {
			select {
			case <-ticker.C:
				res.publish(events)
				events = []Event{}
			case ev, ok := <-res.in:
				if !ok {
					break
				}
				events = append(events, ev)

			}
		}
		log.Println("DEBUG: Closing Trottle()")
	}()

	return res
}

// Subscribe subscribes to an Observable and provides callbacks to handle the events it emits and any error or completion event it issues.
func (s *Stream) Subscribe(onValue, onError, onComplete CallbackFunc) *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
		lastValue := Empty()
		for ev := range res.in {
			switch ev.(type) {
			case ErrorEvent:
				if onError != nil {
					go onError(ev)
				}
			case CompleteEvent:
				if onComplete != nil {
					go onComplete(lastValue)
				}
			default:
				if onValue != nil {
					go onValue(ev)
				}
				lastValue = ev
			}
			res.publish(ev)
		}
		log.Println("DEBUG: Closing Subscribe()")
	}()

	return res
}

// Collect
func (s *Stream) Collect() *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
		var event []Event
		for ev := range res.in {
			switch ev.(type) {
			case ErrorEvent:
				res.RLock()
				res.publish(ev)
				res.RUnlock()
			case CompleteEvent:
				res.RLock()
				res.publish(ev)
				res.RUnlock()
			default:
				res.Lock()
				event = append(event, ev)
				res.Unlock()

				res.RLock()
				res.publish(event)
				res.RUnlock()
			}
		}
		log.Println("DEBUG: Closing Collect()")
	}()
	return res
}

type KeyFunc func(ev Event) string

// GroupBy
func (s *Stream) GroupBy(keyfn KeyFunc) *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
		event := make(map[string][]Event)
		for ev := range res.in {
			switch ev.(type) {
			case ErrorEvent:
				res.RLock()
				res.publish(ev)
				res.RUnlock()
			case CompleteEvent:
				res.RLock()
				res.publish(ev)
				res.RUnlock()
			default:
				key := keyfn(ev)
				res.Lock()
				event[key] = append(event[key], ev)
				res.Unlock()
				res.RLock()
				res.publish(event)
				res.RUnlock()
			}
		}
		log.Println("DEBUG: Closing GroupBy()")
	}()
	return res
}

func (s *Stream) Distinct(keyfn KeyFunc) *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
		event := make(map[string]Event)
		for ev := range res.in {
			switch ev.(type) {
			case ErrorEvent:
				res.RLock()
				res.publish(ev)
				res.RUnlock()
			case CompleteEvent:
				res.RLock()
				res.publish(ev)
				res.RUnlock()
			default:
				key := keyfn(ev)
				res.Lock()
				event[key] = ev
				res.Unlock()
				res.RLock()
				res.publish(event)
				res.RUnlock()
			}
		}
		log.Println("DEBUG: Closing Distinct()")
	}()
	return res
}

// Signal is a value that changes over time.
type Signal struct {
	sync.RWMutex
	parent      *Stream
	event       Event
	initVal     Event
	initialized bool
	values      observers
	errors      observers
	completed   observers
}

func newSignal() *Signal {
	return &Signal{
		parent:      newStream(),
		initialized: true,
		values:      make(observers),
		errors:      make(observers),
		completed:   make(observers),
	}
}

func (s *Signal) Stream() *Stream {
	s.RLock()
	defer s.RUnlock()
	return s.parent
}

func (s *Signal) Close() {
	s.parent.root.Close()
}

// Hold generates a Signal from a stream.
// The signal always holds the last event from the stream.
func (s *Stream) Hold(initVal Event) *Signal {
	res := newSignal()
	res.initVal = initVal
	res.event = res.initVal

	s.subscribe(res.parent)

	go func() {
		for ev := range res.parent.in {
			switch ev.(type) {
			case ErrorEvent:
				res.RLock()
				res.parent.publish(ev)
				res.errors.publish(ev)
				res.RUnlock()
			case CompleteEvent:
				res.RLock()
				res.parent.publish(ev)
				res.completed.publish(res.event)
				res.RUnlock()
			default:
				res.Lock()
				res.initialized = true
				res.event = ev
				res.Unlock()
				res.RLock()
				res.parent.publish(res.event)
				res.values.publish(res.event)
				res.RUnlock()
			}
		}
		log.Println("DEBUG: Closing Hold()")
	}()
	return res
}

// GroupBy is deprecated
// func (s *Stream) GroupBy(keyfn KeyFunc) *Signal {
// 	res := newSignal()
// 	res.initVal = make(map[string][]Event)
// 	res.event = res.initVal

// 	s.subscribe(res.parent)

// 	go func() {
// 		for ev := range res.parent.in {
// 			switch ev.(type) {
// 			case ErrorEvent:
// 				res.RLock()
// 				res.parent.publish(ev)
// 				res.errors.publish(ev)
// 				res.RUnlock()
// 			case CompleteEvent:
// 				res.RLock()
// 				res.parent.publish(ev)
// 				res.completed.publish(res.event)
// 				res.RUnlock()
// 			default:
// 				key := keyfn(ev)
// 				res.Lock()
// 				res.initialized = true

type Subscription *CallbackFunc
type observers map[Subscription]CallbackFunc

func (o *observers) subscribe(callbackfn CallbackFunc) Subscription {
	sub := &callbackfn
	(*o)[sub] = callbackfn
	return sub
}

func (o *observers) unsubscribe(sub Subscription) {
	delete(*o, sub)
}

func (o *observers) publish(ev Event) {
	for _, observer := range *o {
		go observer(ev)
	}

}

func (s *Signal) Reset() {
	s.Lock()
	defer s.Unlock()
	s.initialized = false
	s.event = s.initVal
}

// Value returns the current value of the Signal.
func (s *Signal) Value() Event {
	s.RLock()
	defer s.RUnlock()
	if s.initialized {
		return s.event
	}
	return s.initVal
}

// CallbackFunc is the function signature used by OnValue().
type CallbackFunc func(Event)

// OnValue registers callback functions that are called each time
// the Signal is updated.
func (s *Signal) OnValue(callbackfn CallbackFunc) Subscription {
	s.Lock()
	defer s.Unlock()
	return s.values.subscribe(callbackfn)
}

// OnError registers callback functions that are called each time
// the Signal receives an error event.
func (s *Signal) OnError(callbackfn CallbackFunc) Subscription {
	s.Lock()
	defer s.Unlock()
	return s.errors.subscribe(callbackfn)
}

// OnComplete registers callback functions that are called
// when there are no more values to expect.
func (s *Signal) OnComplete(callbackfn CallbackFunc) Subscription {
	s.Lock()
	defer s.Unlock()
	return s.completed.subscribe(callbackfn)
}

// Unsubscribe does its work to unsubscribe callback functions
// from On{Value | Error | Complete}.
func (s *Signal) Unsubscribe(sub Subscription) {
	s.Lock()
	defer s.Unlock()
	s.values.unsubscribe(sub)
	s.errors.unsubscribe(sub)
	s.completed.unsubscribe(sub)
}
