package daisychain

import (
	"log"
	"runtime"
	"sync"
)

type Event interface{}
type Events []Event

func (events *Events) add(ev Event) {
	*events = append(*events, ev)
}

func (events Events) find(keyfn KeyFunc) (int, bool) {
	var found bool
	var index int = -1
	for i, event := range events {
		if keyfn(event) {
			found = true
			index = i
			break
		}
	}

	return index, found
}

func remove(events Events, keyfn KeyFunc) Events {
	if index, found := events.find(keyfn); found {
		events = append(events[:index], events[index+1:]...)
	} else {
		log.Println("Warning: No match for update")
	}
	return events
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
		child.send(ev)
	}
}

func (s *Stream) propagate(events Events) {
	s.RLock()
	defer s.RUnlock()
	for child, _ := range s.subs {
		child.update(events)
	}
}

type Stream struct {
	sync.RWMutex
	in          chan Event
	recalculate chan Events
	quit        chan bool
	subs        subscribers
}

type KeyFunc func(Event) bool

type replacement struct {
	newEvent Event
	keyfn    KeyFunc
}

type Sink struct {
	*Stream
}

type UpdateableSink struct {
	*Sink
	recalculate chan replacement
}

func newStream() *Stream {
	return &Stream{
		in:          make(chan Event),
		recalculate: make(chan Events),
		quit:        make(chan bool),
		subs:        make(subscribers),
	}
}

func newSink() *Sink {
	return &Sink{
		Stream: newStream(),
	}
}

func newUpdateableSink() *UpdateableSink {
	return &UpdateableSink{
		Sink:        newSink(),
		recalculate: make(chan replacement),
	}
}

func sinkFinalizer(s *Sink) {
	s.close()
}

func updateableSinkFinalizer(s *UpdateableSink) {
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

func NewUpdateableSink() *UpdateableSink {
	s := newUpdateableSink()
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
			case rplmt, ok := <-s.recalculate:
				if !ok {
					log.Println("Error recalc sink:", rplmt, ok)
					break Loop
				}
				events = remove(events, rplmt.keyfn)
				events.add(rplmt.newEvent)
				s.propagate(events)
			case <-s.quit:
				break Loop
			}
		}
	}()

	runtime.SetFinalizer(s, updateableSinkFinalizer)

	return s
}

func (s *Sink) Send(ev Event) {
	s.in <- ev
}

func (s *UpdateableSink) Update(keyfn KeyFunc, ev Event) {
	s.recalculate <- replacement{
		newEvent: ev,
		keyfn:    keyfn,
	}
}

func (s *Stream) send(ev Event) {
	s.in <- ev
}

func (s *Stream) update(events Events) {
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
			case newEvents, ok := <-res.recalculate:
				if !ok {
					break Loop
				}

				if newEvents != nil {
					var recalculation Events
					for _, ev := range newEvents {
						val := mapfn(ev)
						recalculation.add(val)
					}
					res.propagate(recalculation)
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
			case newEvents, ok := <-res.recalculate:
				if !ok {
					break Loop
				}
				if newEvents != nil {
					val = init
					var recalculation Events
					for _, ev := range newEvents {
						val = reducefn(val, ev)
						recalculation.add(val)
					}
					res.propagate(recalculation)
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
			case newEvents, ok := <-res.recalculate:
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
					res.propagate(recalculation)
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
			case newEvents, ok := <-res.recalculate:
				if !ok {
					break Loop
				}
				if newEvents != nil {
					var recalculation Events
					for _, ev := range newEvents {
						if ev != nil {
							recalculation.add(ev)
						}
					}
					res.propagate(recalculation)
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
				res.events.add(ev)
				res.Unlock()
				if res.callbackfn != nil {
					go res.callbackfn(ev)
				}
			case newEvents, ok := <-res.parent.recalculate:
				if !ok {
					break Loop
				}
				var lastEvent Event
				if newEvents != nil {
					res.Lock()
					res.events = newEvents
					lastEvent = res.events[len(res.events):]
					res.Unlock()
					if res.callbackfn != nil {
						go res.callbackfn(lastEvent)
					}
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
	parent     *Stream
	events     Events
	initVal    Event
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
	if l := len(s.events); l > 0 {
		return s.events[l-1]
	}
	return s.initVal // if there is none, take initVal
}
