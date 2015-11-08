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
	out  chan chan Event
	quit chan bool
	subs subscribers
}

func newStream() *Stream {
	return &Stream{
		in:   make(chan Event),
		out:  make(chan chan Event),
		quit: make(chan bool),
		subs: make(subscribers),
	}
}

type EmitterFunc func(chan Event)

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
			case out := <-s.out:
				for _, ev := range events {
					out <- ev
				}
				close(out)
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

func (s *Stream) Events() Events {
	var events Events
	in := make(chan Event)
	s.out <- in
	for ev := range in {
		events.add(ev)
	}
	return events
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
			case out := <-res.out:
				for _, ev := range events {
					out <- ev
				}
				close(out)
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
			case out := <-res.out:
				for _, ev := range events {
					out <- ev
				}
				close(out)
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
			case out := <-res.out:
				for _, ev := range events {
					out <- ev
				}
				close(out)
			case <-res.quit:
				break Loop
			}
		}
	}()

	return res
}

type EventFunc func(Event)

func (s *Stream) OnEvent(eventfn EventFunc) *Stream {
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
				go eventfn(ev)
			case <-res.quit:
				break Loop
			}
		}
	}()

	return res
}
