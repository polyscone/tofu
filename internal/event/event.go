package event

import "context"

// Handler represents any function that is capable of handling
// dispatched events.
// The underlying type is set to any to allow for any signatures, but a handler
// should be of the signature func(<event>).
// That is, it should accept an event type to handle, and have no returns.
type Handler any

// Event represents any data that should be considered an event.
// This can be anything from simple primitives to structures.
type Event any

// AnyHandler represents a handler that can be called for any event type.
type AnyHandler func(ctx context.Context, evt Event)

// FallbackHandler represents a handler that can be called when no specific
// handler is listening for an event type.
// That is, it can be used as a catch-all handler for any events that are not
// handled by more explicit handlers.
type FallbackHandler func(ctx context.Context, evt Event)

// Broker defines a type that can register listeners and dispatch/flush events.
type Broker interface {
	Listen(handler Handler)
	ListenAny(handler AnyHandler)
	ListenFallback(handler FallbackHandler)

	Clear()
	Dispatch(ctx context.Context, evt Event)
	Flush(ctx context.Context, queues ...Queue) (nFlushed int)
}

// Queue defines a type that can queue up events, and then either flush them
// using a broker, or clear them.
type Queue interface {
	Enqueue(evt Event)
	Flush(ctx context.Context, broker Broker) (nFlushed int)
	Clear() (nCleared int)
}
