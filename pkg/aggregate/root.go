package aggregate

import "github.com/polyscone/tofu/pkg/event"

// Root is used to represent aggregate roots in a domain.
// It includes an embedded event queue which can be flushed
// at the appropriate time.
type Root struct {
	Events event.MemoryQueue
}
