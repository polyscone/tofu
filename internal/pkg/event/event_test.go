package event_test

import (
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

		var fooHandled bool
		broker.Listen(func(evt FooEvent) {
			fooHandled = true

			if got := evt; got == FooEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		var barHandled bool
		broker.Listen(func(evt BarEvent) {
			barHandled = true

			if got := evt; got == BarEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		var barPtrHandled bool
		broker.Listen(func(evt *BarEvent) {
			barPtrHandled = true

			if got := *evt; got == BarEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		broker.ListenFallback(func(evt event.Event) {
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
		broker.Listen(func(evt FooEvent) {
			fooHandledCount++

			if got := evt; got == FooEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		broker.ListenAny(func(evt event.Event) {
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
}
