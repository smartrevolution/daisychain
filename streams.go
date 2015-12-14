package daisychain

import (
	"runtime"
	"sync"
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

func NewErrorEvent(msg string) Event {
	return Error{msg: msg}
}

type Stream struct {
	sync.RWMutex
	in   chan Event
	quit chan bool
	subs subscribers
}

type Sink struct {
	*Stream
}

func newStream() *Stream {
	return &Stream{
		in:   make(chan Event),
		quit: make(chan bool),
		subs: make(subscribers),
	}
}

func newSink() *Sink {
	return &Sink{
		Stream: newStream(),
	}
}

func sinkFinalizer(s *Sink) {
	s.close()
}

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

func (s *Sink) Send(ev Event) {
	s.in <- ev
}

func (s *Stream) send(ev Event) {
	s.in <- ev
}

func (s *Stream) close() {
	s.quit <- true
	for child := range s.subs {
		s.unsubscribe(child)
		child.close()
	}
}

type MapFunc func(Event) Event

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
					val := mapfn(ev)
					res.publish(val)
				}
			case <-res.quit:
				break Loop
			}
		}
	}()

	return res
}

type ReduceFunc func(left, right Event) Event

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

type FilterFunc func(Event) bool

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

type Signal struct {
	sync.RWMutex
	parent      *Stream
	event       Event
	initVal     Event
	initialized bool
	callbackfn  CallbackFunc
}

func (s *Signal) Value() Event {
	s.RLock()
	defer s.RUnlock()
	if s.initialized {
		return s.event
	}
	return s.initVal
}

type CallbackFunc func(Event)

func (s *Signal) OnValue(callbackfn CallbackFunc) {
	s.Lock()
	defer s.Unlock()
	s.callbackfn = callbackfn
}
