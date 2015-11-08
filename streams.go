package streams

import (
	"sync"
)

type Event interface{}
type Events []Event

func (ev *Events) add(e Event) {
	*ev = append(*ev, e)
}

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
		child.Send(ev)
	}
}

type Stream struct {
	sync.RWMutex
	in   chan Event
	quit chan bool
	subs subscribers
}

func newStream() *Stream {
	return &Stream{
		in:   make(chan Event),
		quit: make(chan bool),
		subs: make(subscribers),
	}
}

func NewStream() *Stream {
	s := newStream()
	go func() {
		var events Events
	Loop:
		for {
			select {
			case ev, ok := <-s.in:
				if !ok {
					break Loop
				}
				events.add(ev)
				s.publish(ev)
			case <-s.quit:
				break Loop
			}
		}
	}()

	return s
}

func (s *Stream) Send(ev Event) {
	s.in <- ev
}

func (s *Stream) Close() {
	for child := range s.subs {
		s.unsubscribe(child)
		child.Close()
	}
	s.quit <- true
}

type MapFunc func(Event) Event

func (s *Stream) Map(mapfn MapFunc) *Stream {
	res := newStream()
	s.subscribe(res)

	go func() {
		var events Events
	Loop:
		for {
			select {
			case ev, ok := <-res.in:
				if !ok {
					break Loop
				}
				if ev != nil {
					val := mapfn(ev)
					events.add(val)
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
		var events Events
	Loop:
		for {
			select {
			case ev, ok := <-res.in:
				if !ok {
					break Loop
				}
				if ev != nil {
					val = reducefn(val, ev)
					events.add(val)
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
		var events Events
	Loop:
		for {
			select {
			case ev, ok := <-res.in:
				if !ok {
					break Loop
				}
				if ev != nil && filterfn(ev) {
					events.add(ev)
					res.publish(ev)
				}
			case <-res.quit:
				break Loop
			}
		}
	}()

	return res
}

type Signal struct {
	sync.RWMutex
	parent     *Stream
	events     Events
	callbackfn CallbackFunc
}

func (s *Signal) Events() Events {
	s.RLock()
	defer s.RUnlock()
	return s.events
}

type CallbackFunc func(Event)

func (s *Signal) OnValue(callbackfn CallbackFunc) {
	s.Lock()
	defer s.Unlock()
	s.callbackfn = callbackfn
}

func (s *Stream) Hold() *Signal {
	res := &Signal{
		parent: newStream(),
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
				res.events.add(ev)
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
