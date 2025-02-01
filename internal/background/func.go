package background

import (
	"log/slog"
	"sync"
)

var wg sync.WaitGroup

func Wait() {
	wg.Wait()
}

func Go(fn func()) {
	wg.Add(1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("background goroutine panic", "recover", r)
			}

			wg.Done()
		}()

		fn()
	}()
}

func GoUnawaited(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("background goroutine panic", "recover", r)
			}
		}()

		fn()
	}()
}
