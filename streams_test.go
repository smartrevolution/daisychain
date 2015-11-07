package streams

import (
	"sync"
	"testing"
)

func TestOnEvent(t *testing.T) {
	//GIVEN
	s0 := NewStream()
	count := 0
	var mutex sync.Mutex
	_ = s0.OnEvent(func(ev Event) {
		mutex.Lock()
		defer mutex.Unlock()
		count++
	})

	//WHEN
	var wg sync.WaitGroup
	wg.Add(1)
	func() {
		for i := 0; i < 10; i++ {
			s0.Send(i)
		}
		wg.Done()
	}()
	wg.Wait()
	//s1.Close()

	//THEN
	if count != 10 {
		t.Log(s0.Events())
		t.Error("Expected 10, Got", count)
	}
	s0.Close()
}

func TestChaining(t *testing.T) {
	//GIVEN
	s0 := NewStream()

	//WHEN
	s1 := s0.Map(func(ev Event) Event {
		return ev.(int) * 2
	})

	s2 := s1.Reduce(func(left, right Event) Event {
		return left.(int) + right.(int)
	}, 0)

	s3 := s2.Filter(func(ev Event) bool {
		return ev.(int) > 50
	})

	var wg sync.WaitGroup
	wg.Add(1)
	func() {
		for i := 0; i < 10; i++ {
			s0.Send(i)
		}
		wg.Done()
	}()
	wg.Wait()

	n0 := s0.Events()
	t.Log("n0", n0)
	n1 := s1.Events()
	t.Log("n1", n1)
	n2 := s2.Events()
	t.Log("n2", n2)
	n3 := s3.Events()
	t.Log("n3", n3)
	s0.Close()

	//THEN
	if len(n0) != 10 {
		t.Error(n0)
	}
	if len(n1) != 10 {
		t.Error(n1)
	}
	if len(n2) != 10 {
		t.Error(n2)
	}
	if len(n3) != 3 {
		t.Error(n3)
	}

}
