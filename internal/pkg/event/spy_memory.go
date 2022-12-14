package event

// SpyMemoryBroker implements a simple in-memory spy event broker intended to
// be used in developer tests that need to verify events have been dispatched.
type SpyMemoryBroker struct {
	*MemoryBroker
	Dispatched bool
}

// NewSpyMemoryBroker returns a new in-memory spy event broker.
func NewSpyMemoryBroker() *SpyMemoryBroker {
	return &SpyMemoryBroker{MemoryBroker: NewMemoryBroker()}
}

// Dispatch will dispatch the given event through an in-memory broker and record
// that an event has been dispatched.
func (sb *SpyMemoryBroker) Dispatch(evt Event) {
	sb.Dispatched = true

	sb.MemoryBroker.Dispatch(evt)
}
