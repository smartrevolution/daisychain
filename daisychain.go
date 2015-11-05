package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

type Event interface{}
type Events []Event

func (ev *Events) add(e Event) {
	*ev = append(*ev, e)
}

type subscribers map[*Stream]interface{}

func (subs subscribers) subscribe(s *Stream) {
	subs[s] = struct{}{}
}

func (subs subscribers) unsubscribe(s *Stream) {
	delete(subs, s)
	close(s.pipe)
}

func (subs subscribers) publish(ev Event) {
	for s := range subs {
		s.pipe <- ev
	}
}

type Stream struct {
	in   chan Event
	pipe chan Event
	subs subscribers
}

type EmitterFunc func(chan Event, *sync.WaitGroup)

func NewStream() *Stream {
	var empty EmitterFunc = func(e chan Event, wg *sync.WaitGroup) {
		wg.Done()
	}
	return NewStreamWith(empty)
}

func NewStreamWith(emitterfn EmitterFunc) *Stream {
	in := make(chan Event)
	pipe := make(chan Event)

	s := &Stream{
		in:   in,
		pipe: pipe,
		subs: make(subscribers),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		var events Events
		for {
			select {
			case ev := <-in:
				events.add(ev)
				s.subs.publish(ev)
			}
		}
	}()

	go emitterfn(in, &wg)
	wg.Wait()

	return s
}

type MapFunc func(Event) Event

func (s *Stream) Map(mapfn MapFunc) *Stream {
	res := NewStream()
	s.subs.subscribe(res)

	go func() {
		for ev := range res.pipe {
			res.in <- mapfn(ev)
		}

	}()
	return res
}

type ReduceFunc func(left, right Event) Event

func (s *Stream) Reduce(reducefn ReduceFunc, init Event) *Stream {
	res := NewStream()
	s.subs.subscribe(res)

	runtime.SetFinalizer(res, func(s *Stream) {
		s.subs.unsubscribe(s)
	})

	go func() {
		value := init
		for {
			select {
			case ev := <-res.pipe:
				value = reducefn(value, ev)
				res.in <- value
			}
		}
	}()

	return res
}

type FilterFunc func(Event) bool

func (s *Stream) Filter(filterfn FilterFunc) *Stream {
	res := NewStream()
	s.subs.subscribe(res)

	go func() {
		for ev := range res.pipe {
			if filterfn(ev) {
				res.in <- ev
			}
		}

	}()

	return res
}

type SampleFunc func(Event)

func (s *Stream) Sample(samplefn SampleFunc) {
	res := NewStream()
	s.subs.subscribe(res)

	go func() {
		for ev := range res.pipe {
			samplefn(ev)
		}
	}()
}

func main() {
	s0 := NewStreamWith(func(out chan Event, wg *sync.WaitGroup) {
		t := time.NewTicker(1 * time.Second)
		wg.Done()
		for {
			select {
			case _ = <-t.C:
				out <- rand.Intn(10)
			}
		}
	})

	s1 := s0.Map(func(ev Event) Event {
		return ev.(int) * 2
	})

	s2 := s1.Reduce(func(left, right Event) Event {
		return left.(int) + right.(int)
	}, 0)

	s3 := s2.Filter(func(ev Event) bool {
		return ev.(int) > 50
	})

	s0.Sample(func(ev Event) {
		fmt.Println("s0", ev)
	})
	s1.Sample(func(ev Event) {
		fmt.Println("s1", ev)
	})
	s2.Sample(func(ev Event) {
		fmt.Println("s2", ev)
	})
	s3.Sample(func(ev Event) {
		fmt.Println("s3", ev)
	})

	time.Sleep(10 * time.Second)
}
