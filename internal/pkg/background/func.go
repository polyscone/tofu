package background

import "log/slog"

func Go(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("background goroutine panic", "recover", r)
			}
		}()

		fn()
	}()
}
