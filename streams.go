package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
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

type EmitterFunc func(chan Event)

func NewStream() *Stream {
	s := &Stream{
		in:   make(chan Event),
		quit: make(chan bool),
		subs: make(subscribers),
	}

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
	res := &Stream{
		in:   make(chan Event),
		quit: make(chan bool),
		subs: make(subscribers),
	}

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
	res := &Stream{
		in:   make(chan Event),
		quit: make(chan bool),
		subs: make(subscribers),
	}

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
				val = reducefn(val, ev)
				events.add(val)
				res.publish(val)
			case <-res.quit:
				break Loop
			}
		}
	}()

	return res
}

type FilterFunc func(Event) bool

func (s *Stream) Filter(filterfn FilterFunc) *Stream {
	res := &Stream{
		in:   make(chan Event),
		quit: make(chan bool),
		subs: make(subscribers),
	}

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
				if filterfn(ev) {
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

// type SampleFunc func(Event)

// func (s *Stream) Sample(samplefn SampleFunc) {
// 	res := NewStream()
// 	s.subscribe(res)

// 	runtime.SetFinalizer(res, func(s *Stream) {
// 		s.unsubscribe(s)
// 		fmt.Println("FINALIZED")
// 	})

// 	go func() {
// 		for ev := range res.pipe {
// 			samplefn(ev)
// 		}
// 	}()
// }

func setup() {

	quit := make(chan bool)
	s0 := NewStream()

	s1 := s0.Map(func(ev Event) Event {
		return ev.(int) * 2
	})

	s2 := s1.Reduce(func(left, right Event) Event {
		return left.(int) + right.(int)
	}, 0)

	_ = s2.Filter(func(ev Event) bool {
		return ev.(int) > 50
	})

	emitter := func() {
		t := time.NewTicker(10 * time.Millisecond)
		for {
			select {
			case _ = <-t.C:
				val := rand.Intn(10)
				s0.Send(val)
			case <-quit:
				fmt.Println("Suicide")
				return
			}
		}
	}

	go emitter()
	time.Sleep(250 * time.Millisecond)
	s0.Close()
	quit <- true
	time.Sleep(time.Second)
}

func main() {
	setup()
}
