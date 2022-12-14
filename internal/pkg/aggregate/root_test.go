package aggregate_test

import (
	"testing"

	"github.com/polyscone/tofu/internal/pkg/aggregate"
	"github.com/polyscone/tofu/internal/pkg/event"
)

func TestAggregateRoot(t *testing.T) {
	var root aggregate.Root
	if _, ok := any(&root.Events).(event.Queue); !ok {
		t.Error("want aggregate.Root to implement event.Queue")
	}
}
