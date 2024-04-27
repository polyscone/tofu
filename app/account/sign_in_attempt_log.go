package account

import (
	"time"

	"github.com/polyscone/tofu/pkg/aggregate"
)

type SignInAttemptLog struct {
	aggregate.Root

	Email         string
	Attempts      int
	LastAttemptAt time.Time
}
