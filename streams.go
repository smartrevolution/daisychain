package daisychain

import (
	"fmt"
	"log"
	"runtime"
	"sync"
)

type Event interface{}
type Events []Event

func (ev *Events) add(e Event) {
	*ev = append(*ev, e)
}

func (ev Events) indexof(e Event) (int, bool) {
	return 0, true //FIXME: Find always the first element...
}

func remove(ev Event, events Events) Events {
	if index, found := events.indexof(ev); found {
		events = append(events[:index], events[index+1:]...) //FIXME: doesn't work on last element
	} else {
		log.Println("Warning: tried to remove no-existing event:", ev)
	}
	return events
}

// func (ev *Events) remove(e Event) {
// 	if index, found := ev.indexof(e); found {
// 		*ev = append(*ev[:index], *ev[index+1:]...) //FIXME: doesn't work on last element
// 	} else {
// 		log.Println("Warning: tried to remove no-existing event:", e)
// 	}
// }

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

func (s *Stream) propagate(events Events) {
	s.RLock()
	defer s.RUnlock()
	for child, _ := range s.subs {
		child.Recalculate(events)
	}
}

type Stream struct {
	sync.RWMutex
	in          chan Event
	recalculate chan Events
	quit        chan bool
	subs        subscribers
}

func newStream() *Stream {
	return &Stream{
		in:          make(chan Event),
		recalculate: make(chan Events),
		quit:        make(chan bool),
		subs:        make(subscribers),
	}
}

func finalizer(s *Stream) {
	fmt.Println("Cleanup", s)
	s.close()
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
			case ev, ok := <-s.recalculate:
				if !ok {
					break Loop
				}

				if ev != nil {
					events = remove(ev, events)
					events.add(ev)
					s.propagate(events)
				}
			case <-s.quit:
				break Loop
			}
		}
	}()

	runtime.SetFinalizer(s, finalizer)

	return s
}

func (s *Stream) Send(ev Event) {
	s.in <- ev
}

func (s *Stream) Update(ev Event) {
	var events Events
	events.add(ev)
	s.in <- events
}

func (s *Stream) Recalculate(events Events) {
	s.recalculate <- events
}

func (s *Stream) close() {
	for child := range s.subs {
		s.unsubscribe(child)
		child.close()
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
			case newEvents, ok := <-s.recalculate:
				if !ok {
					break Loop
				}

				if newEvents != nil {
					var recalculation Events
					for _, ev := range newEvents {
						val := mapfn(ev)
						recalculation.add(val)
					}
					events = recalculation
					s.propagate(events)
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
			case newEvents, ok := <-s.recalculate:
				if !ok {
					break Loop
				}
				if newEvents != nil {
					var recalculation Events
					for _, ev := range newEvents {
						val = reducefn(val, ev)
						recalculation.add(val)
					}
					events = recalculation
					s.propagate(events)
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
			case newEvents, ok := <-s.recalculate:
				if !ok {
					break Loop
				}
				if newEvents != nil {
					var recalculation Events
					for _, ev := range newEvents {
						if ev != nil && filterfn(ev) {
							recalculation.add(ev)
						}
					}
					events = recalculation
					s.propagate(events)
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

func (s *Signal) Get(filterfn FilterFunc) (Event, bool) {
	events := s.Find(filterfn)
	if ok := len(events) > 0; ok {
		return events[0], ok
	}
	return nil, false
}

func (s *Signal) Find(filterfn FilterFunc) Events {
	s.RLock()
	defer s.RUnlock()
	var res Events
	for _, ev := range s.events {
		if filterfn(ev) {
			res.add(ev)
		}
	}
	return res
}

func (s *Signal) Last() Event {
	s.RLock()
	defer s.RUnlock()
	return s.events[len(s.events)-1]
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
