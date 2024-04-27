package event_test

import (
	"testing"

	"github.com/polyscone/tofu/pkg/event"
)

func TestMemoryBrokerAndQueue(t *testing.T) {
	newBroker := func() event.Broker { return event.NewMemoryBroker() }
	newQueue := func() event.Queue { return &event.MemoryQueue{} }

	runTests(t, newBroker, newQueue)
}
