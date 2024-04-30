package session_test

import (
	"testing"

	"github.com/polyscone/tofu/session"
)

func TestSessionManager(t *testing.T) {
	t.Run("in-memory json repo", func(t *testing.T) {
		for _, useNumber := range []bool{false, true} {
			session.TestManager(t, func() session.ReadWriter {
				return session.NewJSONMemoryRepo(useNumber)
			})
		}
	})
}
