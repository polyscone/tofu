package command_test

import (
	"testing"

	"github.com/polyscone/tofu/internal/pkg/command"
)

func TestMemoryBus(t *testing.T) {
	newBus := func() command.Bus { return command.NewMemoryBus() }

	runTests(t, newBus)
}
