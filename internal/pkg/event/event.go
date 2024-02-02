package event

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
type AnyHandler func(evt Event)

// FallbackHandler represents a handler that can be called when no specific
// handler is listening for an event type.
// That is, it can be used as a catch-all handler for any events that are not
// handled by more explicit handlers.
type FallbackHandler func(evt Event)

// Broker defines a type that can register listeners and dispatch/flush events.
// Listen methods must expect to be eventually consistent, though an implementation
// may make them immediately consistent if needed.
// Any ListenImmediate methods must guarantee immediate consistency within the system.
type Broker interface {
	Listen(handler Handler)
	ListenAny(handler AnyHandler)
	ListenFallback(handler FallbackHandler)

	ListenImmediate(handler Handler)
	ListenImmediateAny(handler AnyHandler)
	ListenImmediateFallback(handler FallbackHandler)

	Clear()
	Dispatch(evt Event)
	Flush(queues ...Queue) (nFlushed int)
}

// Queue defines a type that can queue up events, and then either flush them
// using a broker, or clear them.
type Queue interface {
	Enqueue(evt Event)
	Flush(broker Broker) (nFlushed int)
	Clear() (nCleared int)
}
