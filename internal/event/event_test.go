package event_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/event"
)

func runTests(t *testing.T, newBroker func() event.Broker, newQueue func() event.Queue) {
	ctx := context.Background()

	t.Run("valid events and handlers", func(t *testing.T) {
		type FooEvent int
		type BarEvent int

		broker := newBroker()
		queue1 := newQueue()
		queue2 := newQueue()

		var wg sync.WaitGroup

		var fooHandledCount atomic.Int64
		broker.Listen(func(ctx context.Context, data FooEvent, createdAt time.Time) {
			defer wg.Done()

			fooHandledCount.Add(1)

			if got := data; got == FooEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		var barHandledCount atomic.Int64
		broker.Listen(func(ctx context.Context, data BarEvent, createdAt time.Time) {
			defer wg.Done()

			barHandledCount.Add(1)

			if got := data; got == BarEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		var barPtrHandledCount atomic.Int64
		broker.Listen(func(ctx context.Context, data *BarEvent, createdAt time.Time) {
			defer wg.Done()

			barPtrHandledCount.Add(1)

			if got := *data; got == BarEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		broker.ListenFallback(func(ctx context.Context, data any, createdAt time.Time) {
			t.Errorf("unexpected %T was dispatched", data)
		})

		barEvent := BarEvent(3)

		wg.Add(3)
		{
			queue1.Enqueue(FooEvent(1))
			queue1.Enqueue(BarEvent(2))
			queue1.Enqueue(&barEvent)

			if want, got := 3, queue1.Flush(ctx, broker); want != got {
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
		if want, got := 0, queue1.Flush(ctx, broker); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		wg.Add(3)
		{
			queue1.Enqueue(FooEvent(1))
			queue1.Enqueue(BarEvent(2))
			queue2.Enqueue(BarEvent(3))

			if want, got := 3, broker.Flush(ctx, queue1, queue2); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		}
		wg.Wait()

		broker.Clear()

		fooHandledCount.Store(0)
		queue1.Enqueue(FooEvent(1))

		if want, got := 1, broker.Flush(ctx, queue1); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if fooHandledCount.Load() != 0 {
			t.Errorf("want %T to be unhandled", FooEvent(0))
		}

		broker.Listen(func(ctx context.Context, data FooEvent, createdAt time.Time) {
			defer wg.Done()

			fooHandledCount.Add(1)

			if got := data; got == FooEvent(0) {
				t.Errorf("want non-zero event value; got %v", got)
			}
		})

		broker.ListenAny(func(ctx context.Context, data any, createdAt time.Time) {
			if data, ok := data.(FooEvent); ok {
				defer wg.Done()

				fooHandledCount.Add(1)

				if got := data; got == FooEvent(0) {
					t.Errorf("want non-zero event value; got %v", got)
				}
			}
		})

		wg.Add(2) // Handled in Listen and ListenAny
		{
			queue1.Enqueue(FooEvent(1))

			if want, got := 1, broker.Flush(ctx, queue1); want != got {
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
		broker.Listen(func(ctx context.Context, createdAt time.Time) {})
	})

	t.Run("invalid handlers with too many paramters", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		broker := newBroker()
		broker.Listen(func(ctx context.Context, x, y int, createdAt time.Time) {})
	})

	t.Run("invalid handlers with returns", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		broker := newBroker()
		broker.Listen(func(ctx context.Context, i int, createdAt time.Time) error { return nil })
	})
}
