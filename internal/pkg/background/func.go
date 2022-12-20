package background

import "github.com/polyscone/tofu/internal/pkg/logger"

func Go(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error.Println(r)
			}
		}()

		fn()
	}()
}
