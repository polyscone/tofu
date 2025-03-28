package account

import (
	"time"

	"github.com/polyscone/tofu/internal/aggregate"
)

type SignInAttemptLog struct {
	aggregate.Root

	Email         string
	Attempts      int
	LastAttemptAt time.Time
}
