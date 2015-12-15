package daisychain

import (
	"runtime"
	"sync"
	"time"
)

type Event interface{}

type subscribers map[*Stream]interface{}

func (s *Stream) subscribe(child *Stream) {
	s.Lock()
	defer s.Unlock()
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

type Empty struct {
}

func NewEmptyEvent() Event {
	return Empty{}
}

type Error struct {
	Event
	msg string
}

// NewErrorEvent creates a new Event of type Error
func NewErrorEvent(msg string) Event {
	return Error{msg: msg}
}

// Stream is a stream.
type Stream struct {
	sync.RWMutex
	in   chan Event
	quit chan bool
	subs subscribers
	vars map[string]interface{}
}

func (s *Stream) SetVar(name string, value interface{}) {
	//FIXME: Mutex makes another test blocked?! WTF??!?
	//s.Lock()
	//defer s.Unlock()
	s.vars[name] = value
}

func (s *Stream) Var(name string, init interface{}) (value interface{}) {
	s.RLock()
	defer s.RUnlock()
	value, exists := s.vars[name]
	if !exists {
		value = init
	}
	return
}

// Sink is the first node of a stream.
type Sink struct {
	*Stream
}

// newStream is a constructor for streams.
func newStream() *Stream {
	return &Stream{
		in:   make(chan Event),
		quit: make(chan bool),
		subs: make(subscribers),
		vars: make(map[string]interface{}),
	}
}

// newSink is a helper constructor for sinks.
func newSink() *Sink {
	return &Sink{
		Stream: newStream(),
	}
}

// If the runtime discards a Sink, all depending streams are discarded , too.
func sinkFinalizer(s *Sink) {
	s.close()
}

// NewSink generates a new Sink.
func NewSink() *Sink {
	s := newSink()
	go func() {
	Loop:
		for {
			select {
			case ev, ok := <-s.in:
				if !ok {
					break Loop
				}
				s.publish(ev)
			case <-s.quit:
				break Loop
			}
		}
	}()

	runtime.SetFinalizer(s, sinkFinalizer)

	return s
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
	s.quit <- true
	for child := range s.subs {
		s.unsubscribe(child)
		child.close()
	}
}

// MapFunc is the function signature used by Map.
type MapFunc func(*Stream, Event) Event

// Map passes the return value of its MapFunc down the stream.
func (s *Stream) Map(mapfn MapFunc) *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
	Loop:
		for {
			select {
			case ev, ok := <-res.in:
				if !ok {
					break Loop
				}

				if ev != nil {
					val := mapfn(s, ev)
					res.publish(val)
				}
			case <-res.quit:
				break Loop
			}
		}
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
	Loop:
		for {
			select {
			case ev, ok := <-res.in:
				if !ok {
					break Loop
				}

				if ev != nil {
					val = reducefn(val, ev)
					res.publish(val)
				}
			case <-res.quit:
				break Loop
			}
		}
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
	Loop:
		for {
			select {
			case ev, ok := <-res.in:
				if !ok {
					break Loop
				}
				if ev != nil && filterfn(ev) {
					res.publish(ev)
				}
			case <-res.quit:
				break Loop
			}
		}
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
	Loop:
		for {
			select {
			case ev, ok := <-res.in:
				if !ok {
					break Loop
				}
				if ev != nil {
					res.publish(ev)
				}
			case <-res.quit:
				break Loop
			}
		}
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
	Loop:
		for {
			select {
			case <-ticker.C:
				res.publish(events)
				events = []Event{}
			case ev, ok := <-res.in:
				if !ok {
					break Loop
				}
				events = append(events, ev)
			case <-res.quit:
				break Loop
			}
		}
	}()

	return res
}

// Hold generates a Signal from a stream.
func (s *Stream) Hold(initVal Event) *Signal {
	res := &Signal{
		parent:  newStream(),
		event:   NewEmptyEvent(),
		initVal: initVal,
	}
	s.subscribe(res.parent)

	go func() {
	Loop:
		for {
			select {
			case ev, ok := <-res.parent.in:
				if !ok {
					break Loop
				}
				res.Lock()
				res.initialized = true
				res.event = ev
				res.Unlock()
				if res.callbackfn != nil {
					go res.callbackfn(ev)
				}
			case <-res.parent.quit:
				break Loop
			}
		}
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
	callbackfn  CallbackFunc
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

// OnValue registers a callback function that is called each time
// the Signal is updated.
func (s *Signal) OnValue(callbackfn CallbackFunc) {
	s.Lock()
	defer s.Unlock()
	s.callbackfn = callbackfn
}
