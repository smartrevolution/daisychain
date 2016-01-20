package stream

import (
	"log"
	"runtime"
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

type CompleteEvent struct {
	Event
}

// Complete creates a new Event of type CompleteEvent
// to notify down-stream that no more values will be send.
func Complete() Event {
	return CompleteEvent{}
}

type EmptyEvent struct {
	Event
}

// Empty creates a new Event of type ErrorEvent
// to notify down-stream an empty value.
func Empty() Event {
	return EmptyEvent{}
}

type ErrorEvent struct {
	Event
	msg string
}

// Error creates a new Event of type ErrorEvent.
func Error(msg string) Event {
	return ErrorEvent{msg: msg}
}

// Stream is a stream.
type Stream struct {
	sync.RWMutex
	in   chan Event
	subs subscribers
	root *Sink
}

func (s *Stream) Close() {
	s.root.Close()
}

// Sink is the first node of a stream.
type Sink struct {
	*Stream
}

// newStream is a constructor for streams.
func newStream() *Stream {
	return &Stream{
		in:   make(chan Event),
		subs: make(subscribers),
	}
}

// newSink is a helper constructor for sinks.
func newSink() *Sink {
	sink := &Sink{
		Stream: newStream(),
	}
	sink.root = sink
	return sink
}

// If the runtime discards a Sink, all depending streams are discarded , too.
func sinkFinalizer(s *Sink) {
	log.Println("DEBUG: Called Finalizer")
	s.close()
}

// NewSink generates a new Sink.
func New() *Sink {
	s := newSink()
	runtime.SetFinalizer(s, sinkFinalizer)
	go func() {
		for ev := range s.in {
			s.publish(ev)
		}
		log.Println("DEBUG: Closing Sink")
	}()

	return s
}

func (s *Sink) From(evts ...Event) {
	for ev := range evts {
		s.Send(ev)
	}
}

func (s *Sink) Close() {
	close(s.in)
	s.close()
}

// Send sends an Event from a Sink down the stream
func (s *Sink) Send(ev Event) {
	s.in <- ev
}

// send sends an Event down the stream.
func (s *Stream) send(ev Event) {
	s.in <- ev
}

// close will unsubscribe all childs recursively.
func (s *Stream) close() {
	for child := range s.subs {
		s.unsubscribe(child)
		child.close()
		close(child.in)
	}
}

func isCompleteEvent(ev Event) bool {
	_, isCompleteEvent := ev.(CompleteEvent)
	return isCompleteEvent
}

// MapFunc is the function signature used by Map.
type MapFunc func(Event) Event

// Map passes the return value of its MapFunc down the stream.
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

//ReduceFunc is the function signature used by Reduce().
type ReduceFunc func(left, right Event) Event

// Reduce accumulates the passed events. It starts with the init Event.
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

// Filter only fires en event, when the FilterFunc returns true.
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

// Merge merges two streams into one stream.
// Merge fires an event each time either of
// the input streams fires an event.
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

// Throttle collects events and returns them every d time durations.
// Use type assertion
// val, ok := ev.([]Event)
// to get the collected events from a signal.
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

// Hold generates a Signal from a stream.
// The signal always holds the last event from the stream.
func (s *Stream) Hold(initVal Event) *Signal {
	res := newSignal()
	res.initVal = initVal
	res.event = res.initVal

	s.subscribe(res.parent)

	go func() {
		for ev := range res.parent.in {
			res.Lock()
			switch ev.(type) {
			case ErrorEvent:
				res.errors.publish(ev)
			case CompleteEvent:
				res.completed.publish(ev)
			default:
				res.initialized = true
				res.event = ev
				res.values.publish(res.event)
			}
			res.Unlock()
		}
		log.Println("DEBUG: Closing Hold()")
	}()
	return res
}

// Collect collects all events from a stream.
// It appends these events internally to an []Event,
// which it also passes asynchronuously to the CallbackFunc any time
// a new event arrives.
func (s *Stream) Collect() *Signal {
	res := newSignal()
	res.initVal = []Event{}
	res.event = res.initVal

	s.subscribe(res.parent)

	go func() {
		for ev := range res.parent.in {
			res.Lock()
			switch ev.(type) {
			case ErrorEvent:
				res.errors.publish(ev)
			case CompleteEvent:
				res.completed.publish(ev)
			default:
				res.initialized = true
				res.event = append(res.event.([]Event), ev)
				res.values.publish(res.event)
			}
			res.Unlock()
		}
		log.Println("DEBUG: Closing Collect()")
	}()
	return res
}

type KeyFunc func(ev Event) string
type group map[string][]Event

// GroupBy collects all events under the string key
// that is returned by applying KeyFunc to the event.
func (s *Stream) GroupBy(keyfn KeyFunc) *Signal {
	res := newSignal()
	res.initVal = make(map[string][]Event)
	res.event = res.initVal

	s.subscribe(res.parent)

	go func() {
		for ev := range res.parent.in {
			res.Lock()
			switch ev.(type) {
			case ErrorEvent:
				res.errors.publish(ev)
			case CompleteEvent:
				res.completed.publish(ev)
			default:
				res.initialized = true
				key := keyfn(ev)
				(res.event.(map[string][]Event))[key] = append(res.event.(map[string][]Event)[key], ev)
				res.values.publish(res.event)
			}
			res.Unlock()
		}
		log.Println("DEBUG: Closing GroupBy()")
	}()
	return res
}

// Distinct collects unique events under the string key
// that is returned by applying KeyFunc to the event.
func (s *Stream) Distinct(keyfn KeyFunc) *Signal {
	res := newSignal()
	res.initVal = make(map[string]Event)
	res.event = res.initVal

	s.subscribe(res.parent)

	go func() {
		for ev := range res.parent.in {
			res.Lock()
			switch ev.(type) {
			case ErrorEvent:
				res.errors.publish(ev)
			case CompleteEvent:
				res.completed.publish(ev)
			default:
				res.initialized = true
				key := keyfn(ev)
				(res.event.(map[string]Event))[key] = ev
				res.values.publish(res.event)
			}
			res.Unlock()
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

func (s *Signal) Close() {
	s.parent.root.Close()
}

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
