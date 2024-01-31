package event

import (
	"fmt"
	"reflect"

	"github.com/polyscone/tofu/internal/pkg/background"
)

const (
	anyKey      = ".memorybroker.all.any"
	fallbackKey = ".memorybroker.fallback.any"
)

// MemoryBroker implements an in-memory event broker.
// The ImmediateOnly flag can be set to true for simpler use in tests.
type MemoryBroker struct {
	ImmediateOnly bool
	eventual      map[string][]reflect.Value
	immediate     map[string][]reflect.Value
}

// NewMemoryBroker returns a new in-memory event broker.
func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		eventual:  make(map[string][]reflect.Value),
		immediate: make(map[string][]reflect.Value),
	}
}

// Listen registers a new handler for an event type.
// Multiple handlers for the same event type can be registered.
// Any handlers registered through this method must expect to be eventually consistent.
func (mb *MemoryBroker) Listen(handler Handler) {
	listenerFuncType := reflect.TypeOf(handler)

	if want, got := 1, listenerFuncType.NumIn(); want != got {
		panic(fmt.Sprintf("handler must have %v parameters; got %v", want, got))
	}

	if want, got := 0, listenerFuncType.NumOut(); want != got {
		panic(fmt.Sprintf("handler must have %v returns; got %v", want, got))
	}

	key := eventKey(listenerFuncType.In(0))
	mb.eventual[key] = append(mb.eventual[key], reflect.ValueOf(handler))
}

// ListenAny registers a listener for any events.
// Multiple handlers for the same event type can be registered.
// Any handlers registered through this method must expect to be eventually consistent.
func (mb *MemoryBroker) ListenAny(handler AnyHandler) {
	listenerFuncType := reflect.TypeOf(handler)

	if want, got := 1, listenerFuncType.NumIn(); want != got {
		panic(fmt.Sprintf("handler must have %v parameters; got %v", want, got))
	}

	mb.eventual[anyKey] = append(mb.eventual[anyKey], reflect.ValueOf(handler))
}

// ListenFallback registers a fallback listener for any events that are not
// handled by a more specific handler.
// Multiple handlers for the same event type can be registered.
// Any handlers registered through this method must expect to be eventually consistent.
func (mb *MemoryBroker) ListenFallback(handler FallbackHandler) {
	listenerFuncType := reflect.TypeOf(handler)

	if want, got := 1, listenerFuncType.NumIn(); want != got {
		panic(fmt.Sprintf("handler must have %v parameters; got %v", want, got))
	}

	mb.eventual[fallbackKey] = append(mb.eventual[fallbackKey], reflect.ValueOf(handler))
}

// ListenImmediate registers a new handler for an event type.
// Multiple handlers for the same event type can be registered.
// Any handlers registered through this method are guaranteed to be immediately consistent.
func (mb *MemoryBroker) ListenImmediate(handler Handler) {
	listenerFuncType := reflect.TypeOf(handler)

	if want, got := 1, listenerFuncType.NumIn(); want != got {
		panic(fmt.Sprintf("handler must have %v parameters; got %v", want, got))
	}

	if want, got := 0, listenerFuncType.NumOut(); want != got {
		panic(fmt.Sprintf("handler must have %v returns; got %v", want, got))
	}

	key := eventKey(listenerFuncType.In(0))
	mb.immediate[key] = append(mb.immediate[key], reflect.ValueOf(handler))
}

// ListenImmediateAny registers a listener for any events.
// Multiple handlers for the same event type can be registered.
// Any handlers registered through this method are guaranteed to be immediately consistent.
func (mb *MemoryBroker) ListenImmediateAny(handler AnyHandler) {
	listenerFuncType := reflect.TypeOf(handler)

	if want, got := 1, listenerFuncType.NumIn(); want != got {
		panic(fmt.Sprintf("handler must have %v parameters; got %v", want, got))
	}

	mb.immediate[anyKey] = append(mb.immediate[anyKey], reflect.ValueOf(handler))
}

// ListenImmediateFallback registers a fallback listener for any events that are not
// handled by a more specific handler.
// Multiple handlers for the same event type can be registered.
// Any handlers registered through this method are guaranteed to be immediately consistent.
func (mb *MemoryBroker) ListenImmediateFallback(handler FallbackHandler) {
	listenerFuncType := reflect.TypeOf(handler)

	if want, got := 1, listenerFuncType.NumIn(); want != got {
		panic(fmt.Sprintf("handler must have %v parameters; got %v", want, got))
	}

	mb.immediate[fallbackKey] = append(mb.immediate[fallbackKey], reflect.ValueOf(handler))
}

// Clear removes all registered listeners.
func (mb *MemoryBroker) Clear() {
	mb.eventual = make(map[string][]reflect.Value)
	mb.immediate = make(map[string][]reflect.Value)
}

// Dispatch attempts to dispatch the given event to any handlers registered for
// that event type.
// In the case where no specific handler for the event type is listening it will
// dispatch the event to any fallback handlers.
// If no fallback handlers are registered then the event is ignored.
func (mb *MemoryBroker) Dispatch(evt Event) {
	args := []reflect.Value{reflect.ValueOf(evt)}

	// Immediately consistent event handlers
	{
		key := eventKey(reflect.TypeOf(evt))

		if len(mb.immediate[key]) == 0 {
			key = fallbackKey
		}

		for _, handler := range mb.immediate[key] {
			handler.Call(args)
		}

		for _, handler := range mb.immediate[anyKey] {
			handler.Call(args)
		}
	}

	// Eventually consistent event handlers
	// If the ImmediateOnly flag is set then these behave the same as
	// immediately consistent handlers
	{
		key := eventKey(reflect.TypeOf(evt))

		if len(mb.eventual[key]) == 0 {
			key = fallbackKey
		}

		if mb.ImmediateOnly {
			for _, handler := range mb.eventual[key] {
				handler.Call(args)
			}

			for _, handler := range mb.eventual[anyKey] {
				handler.Call(args)
			}
		} else {
			for _, handler := range mb.eventual[key] {
				background.Go(func() { handler.Call(args) })
			}

			for _, handler := range mb.eventual[anyKey] {
				background.Go(func() { handler.Call(args) })
			}
		}
	}
}

// Flush is a helper method that will flush all of the given queues
// through itself.
func (mb *MemoryBroker) Flush(queues ...Queue) int {
	var n int
	for _, queue := range queues {
		n += queue.Flush(mb)
	}

	return n
}

func eventKey(typ reflect.Type) string {
	var prefix string
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		prefix += "*"
	}

	return prefix + typ.PkgPath() + "." + typ.Name()
}

// MemoryQueue implements a simple in-memory event queue.
type MemoryQueue struct {
	events []Event
}

// Enqueue adds a new event to the end of the event queue.
func (q *MemoryQueue) Enqueue(evt Event) {
	q.events = append(q.events, evt)
}

// Flush dispatches all of the queued events through the given event broker.
func (q *MemoryQueue) Flush(broker Broker) int {
	n := len(q.events)
	for _, evt := range q.events {
		broker.Dispatch(evt)
	}

	q.events = q.events[:0]

	return n
}

// Clear discards all queued events without dispatching them.
func (q *MemoryQueue) Clear() int {
	n := len(q.events)

	q.events = q.events[:0]

	return n
}
