package event_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/event"
)

func runTests(t *testing.T, newBroker func() event.Broker, newQueue func() event.Queue) {
	t.Run("valid events and handlers", func(t *testing.T) {
		type FooEvent int
		type BarEvent int

		broker := newBroker()
		queue1 := newQueue()
		queue2 := newQueue()

		var wg sync.WaitGroup

		var fooHandledCount atomic.Int64
		broker.Listen(func(evt FooEvent) {
			defer wg.Done()

			fooHandledCount.Add(1)

			if got := evt; got == FooEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		var barHandledCount atomic.Int64
		broker.Listen(func(evt BarEvent) {
			defer wg.Done()

			barHandledCount.Add(1)

			if got := evt; got == BarEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		var barPtrHandledCount atomic.Int64
		broker.Listen(func(evt *BarEvent) {
			defer wg.Done()

			barPtrHandledCount.Add(1)

			if got := *evt; got == BarEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		broker.ListenFallback(func(evt event.Event) {
			t.Errorf("unexpected %T was dispatched", evt)
		})

		barEvent := BarEvent(3)

		wg.Add(3)
		{
			queue1.Enqueue(FooEvent(1))
			queue1.Enqueue(BarEvent(2))
			queue1.Enqueue(&barEvent)

			if want, got := 3, queue1.Flush(broker); want != got {
				t.Errorf("want %v events flushed; got %v", want, got)
			}
		}
		wg.Wait()

		if fooHandledCount.Load() == 0 {
			t.Errorf("want %T to be handled", FooEvent(0))
		}
		if barHandledCount.Load() == 0 {
			t.Errorf("want %T to be handled", BarEvent(0))
		}
		if barPtrHandledCount.Load() == 0 {
			t.Errorf("want %T to be handled", &barEvent)
		}

		queue1.Enqueue(FooEvent(1))
		queue1.Enqueue(BarEvent(2))

		if want, got := 2, queue1.Clear(); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 0, queue1.Flush(broker); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		wg.Add(3)
		{
			queue1.Enqueue(FooEvent(1))
			queue1.Enqueue(BarEvent(2))
			queue2.Enqueue(BarEvent(3))

			if want, got := 3, broker.Flush(queue1, queue2); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		}
		wg.Wait()

		broker.Clear()

		fooHandledCount.Store(0)
		queue1.Enqueue(FooEvent(1))

		if want, got := 1, broker.Flush(queue1); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if fooHandledCount.Load() != 0 {
			t.Errorf("want %T to be unhandled", FooEvent(0))
		}

		broker.Listen(func(evt FooEvent) {
			defer wg.Done()

			fooHandledCount.Add(1)

			if got := evt; got == FooEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		broker.ListenAny(func(evt event.Event) {
			if evt, ok := evt.(FooEvent); ok {
				defer wg.Done()

				fooHandledCount.Add(1)

				if got := evt; got == FooEvent(0) {
					t.Errorf("want non-zero event value; got %v", got)
				}
			}
		})

		wg.Add(2) // Handled in Listen and ListenAny
		{
			queue1.Enqueue(FooEvent(1))

			if want, got := 1, broker.Flush(queue1); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		}
		wg.Wait()

		if want, got := int64(2), fooHandledCount.Load(); got != want {
			t.Errorf("want %T to be handled %v times; got %v", FooEvent(0), want, got)
		}

	})

	t.Run("invalid handlers with no paramters", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		broker := newBroker()
		broker.Listen(func() {})
	})

	t.Run("invalid handlers with too many paramters", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		broker := newBroker()
		broker.Listen(func(x, y int) {})
	})

	t.Run("invalid handlers with returns", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		broker := newBroker()
		broker.Listen(func(i int) error { return nil })
	})

	t.Run("valid events and handlers (immediate)", func(t *testing.T) {
		type FooEvent int
		type BarEvent int

		broker := newBroker()
		queue1 := newQueue()
		queue2 := newQueue()

		var fooHandled bool
		broker.ListenImmediate(func(evt FooEvent) {
			fooHandled = true

			if got := evt; got == FooEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		var barHandled bool
		broker.ListenImmediate(func(evt BarEvent) {
			barHandled = true

			if got := evt; got == BarEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		var barPtrHandled bool
		broker.ListenImmediate(func(evt *BarEvent) {
			barPtrHandled = true

			if got := *evt; got == BarEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		broker.ListenImmediateFallback(func(evt event.Event) {
			t.Errorf("unexpected %T was dispatched", evt)
		})

		barEvent := BarEvent(3)

		queue1.Enqueue(FooEvent(1))
		queue1.Enqueue(BarEvent(2))
		queue1.Enqueue(&barEvent)

		if want, got := 3, queue1.Flush(broker); want != got {
			t.Errorf("want %v events flushed; got %v", want, got)
		}
		if !fooHandled {
			t.Errorf("want %T to be handled", FooEvent(0))
		}
		if !barHandled {
			t.Errorf("want %T to be handled", BarEvent(0))
		}
		if !barPtrHandled {
			t.Errorf("want %T to be handled", &barEvent)
		}

		queue1.Enqueue(FooEvent(1))
		queue1.Enqueue(BarEvent(2))

		if want, got := 2, queue1.Clear(); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 0, queue1.Flush(broker); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		queue1.Enqueue(FooEvent(1))
		queue1.Enqueue(BarEvent(2))
		queue2.Enqueue(BarEvent(3))

		if want, got := 3, broker.Flush(queue1, queue2); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		broker.Clear()

		fooHandled = false
		queue1.Enqueue(FooEvent(1))

		if want, got := 1, broker.Flush(queue1); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if fooHandled {
			t.Errorf("want %T to be unhandled", FooEvent(0))
		}

		var fooHandledCount int
		broker.ListenImmediate(func(evt FooEvent) {
			fooHandledCount++

			if got := evt; got == FooEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		broker.ListenImmediateAny(func(evt event.Event) {
			if evt, ok := evt.(FooEvent); ok {
				fooHandledCount++

				if got := evt; got == FooEvent(0) {
					t.Errorf("want non-zero event value; got %v", got)
				}
			}
		})

		queue1.Enqueue(FooEvent(1))

		if want, got := 1, broker.Flush(queue1); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want := 2; fooHandledCount != want {
			t.Errorf("want %T to be handled %v times; got %v", FooEvent(0), want, fooHandledCount)
		}
	})

	t.Run("invalid handlers with no paramters (immediate)", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		broker := newBroker()
		broker.ListenImmediate(func() {})
	})

	t.Run("invalid handlers with too many paramters (immediate)", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		broker := newBroker()
		broker.ListenImmediate(func(x, y int) {})
	})

	t.Run("invalid handlers with returns (immediate)", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		broker := newBroker()
		broker.ListenImmediate(func(i int) error { return nil })
	})
}
