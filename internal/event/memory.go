package event

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

const (
	anyKey      = ".memorybroker.all.any"
	fallbackKey = ".memorybroker.fallback.any"

	handlerParamCount = 3 // (_ context.Context, data any, createdAt time.Time)
)

type MemoryEvent struct {
	data      any
	createdAt time.Time
}

func NewMemoryEvent(data any) MemoryEvent {
	return MemoryEvent{
		data:      data,
		createdAt: time.Now(),
	}
}

func (e MemoryEvent) Data() any {
	return e.data
}

func (e MemoryEvent) CreatedAt() time.Time {
	return e.createdAt
}

// MemoryBroker implements an in-memory event broker.
type MemoryBroker struct {
	handlers map[string][]reflect.Value
}

// NewMemoryBroker returns a new in-memory event broker.
func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		handlers: make(map[string][]reflect.Value),
	}
}

// Listen registers a new handler for an event type.
// Multiple handlers for the same event type can be registered.
func (mb *MemoryBroker) Listen(handler Handler) {
	listenerFuncType := reflect.TypeOf(handler)

	if got := listenerFuncType.NumIn(); handlerParamCount != got {
		panic(fmt.Sprintf("handler must have %v parameters; got %v", handlerParamCount, got))
	}

	const handlerReturnCount = 0
	if got := listenerFuncType.NumOut(); handlerReturnCount != got {
		panic(fmt.Sprintf("handler must have %v returns; got %v", handlerReturnCount, got))
	}

	if want, got := "context.Context", listenerFuncType.In(0).String(); want != got {
		panic(fmt.Sprintf("handler's [0] parameter must be %q; got %q", want, got))
	}

	if want, got := "time.Time", listenerFuncType.In(2).String(); want != got {
		panic(fmt.Sprintf("handler's [2] parameter must be %q; got %q", want, got))
	}

	const handlerKeyParamIndex = 1
	key := eventKey(listenerFuncType.In(handlerKeyParamIndex))
	mb.handlers[key] = append(mb.handlers[key], reflect.ValueOf(handler))
}

// ListenAny registers a listener for any events.
// Multiple handlers for the same event type can be registered.
func (mb *MemoryBroker) ListenAny(handler AnyHandler) {
	listenerFuncType := reflect.TypeOf(handler)

	if got := listenerFuncType.NumIn(); handlerParamCount != got {
		panic(fmt.Sprintf("handler must have %v parameters; got %v", handlerParamCount, got))
	}

	if want, got := "context.Context", listenerFuncType.In(0).String(); want != got {
		panic(fmt.Sprintf("handler's [0] parameter must be %q; got %q", want, got))
	}

	if want, got := "time.Time", listenerFuncType.In(2).String(); want != got {
		panic(fmt.Sprintf("handler's [2] parameter must be %q; got %q", want, got))
	}

	mb.handlers[anyKey] = append(mb.handlers[anyKey], reflect.ValueOf(handler))
}

// ListenFallback registers a fallback listener for any events that are not
// handled by a more specific handler.
// Multiple handlers for the same event type can be registered.
func (mb *MemoryBroker) ListenFallback(handler FallbackHandler) {
	listenerFuncType := reflect.TypeOf(handler)

	if got := listenerFuncType.NumIn(); handlerParamCount != got {
		panic(fmt.Sprintf("handler must have %v parameters; got %v", handlerParamCount, got))
	}

	if want, got := "context.Context", listenerFuncType.In(0).String(); want != got {
		panic(fmt.Sprintf("handler's [0] parameter must be %q; got %q", want, got))
	}

	if want, got := "time.Time", listenerFuncType.In(2).String(); want != got {
		panic(fmt.Sprintf("handler's [2] parameter must be %q; got %q", want, got))
	}

	mb.handlers[fallbackKey] = append(mb.handlers[fallbackKey], reflect.ValueOf(handler))
}

// Clear removes all registered listeners.
func (mb *MemoryBroker) Clear() {
	mb.handlers = make(map[string][]reflect.Value)
}

// Dispatch attempts to dispatch the given event to any handlers registered for
// that event type.
// In the case where no specific handler for the event type is listening it will
// dispatch the event to any fallback handlers.
// If no fallback handlers are registered then the event is ignored.
func (mb *MemoryBroker) Dispatch(ctx context.Context, data any) {
	evt, ok := data.(Event)
	if !ok {
		evt = NewMemoryEvent(data)
	}

	data = evt.Data()
	key := eventKey(reflect.TypeOf(data))
	args := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(data),
		reflect.ValueOf(evt.CreatedAt()),
	}

	if len(mb.handlers[key]) == 0 {
		key = fallbackKey
	}

	for _, handler := range mb.handlers[key] {
		handler.Call(args)
	}

	for _, handler := range mb.handlers[anyKey] {
		handler.Call(args)
	}
}

// Flush is a helper method that will flush all of the given queues
// through itself.
func (mb *MemoryBroker) Flush(ctx context.Context, queues ...Queue) int {
	var n int
	for _, queue := range queues {
		n += queue.Flush(ctx, mb)
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
func (q *MemoryQueue) Enqueue(data any) {
	evt, ok := data.(Event)
	if !ok {
		evt = NewMemoryEvent(data)
	}

	q.events = append(q.events, evt)
}

// Flush dispatches all of the queued events through the given event broker.
func (q *MemoryQueue) Flush(ctx context.Context, broker Broker) int {
	n := len(q.events)
	for _, evt := range q.events {
		broker.Dispatch(ctx, evt)
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
