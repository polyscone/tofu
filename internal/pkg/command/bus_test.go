package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/command"
)

func runTests(t *testing.T, newBus func() command.Bus) {
	t.Run("handler runs on dispatch", func(t *testing.T) {
		ctx := context.Background()
		bus := newBus()

		var hasRun bool
		bus.Register(func(ctx context.Context, cmd string) error {
			hasRun = true

			return nil
		})

		if _, err := bus.Dispatch(ctx, "foo"); err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := true, hasRun; want != got {
			t.Errorf("want %v; got %v", want, got)
		}
	})

	t.Run("handler returns nothing", func(t *testing.T) {
		ctx := context.Background()
		bus := newBus()

		bus.Register(func(ctx context.Context, cmd string) {})

		out, err := bus.Dispatch(ctx, "foo")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if out != nil {
			t.Errorf("want <nil>; got %q", out)
		}
	})

	t.Run("handler returns a value", func(t *testing.T) {
		ctx := context.Background()
		bus := newBus()

		bus.Register(func(ctx context.Context, cmd string) string {
			return "baz"
		})

		out, err := bus.Dispatch(ctx, "foo")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := out.(string), "baz"; want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("handler returns an error", func(t *testing.T) {
		ctx := context.Background()
		bus := newBus()

		bus.Register(func(ctx context.Context, cmd string) error {
			return errors.New(cmd)
		})

		out, err := bus.Dispatch(ctx, "foo")
		if err == nil {
			t.Error("want error; got <nil>")
		}
		if out != nil {
			t.Errorf("want <nil>; got %v", out)
		}
	})

	t.Run("handler returns a value and error", func(t *testing.T) {
		ctx := context.Background()
		bus := newBus()

		bus.Register(func(ctx context.Context, cmd string) (string, error) {
			return "bar", errors.New(cmd)
		})

		out, err := bus.Dispatch(ctx, "foo")
		if err == nil {
			t.Error("want error; got <nil>")
		}
		if want, got := out.(string), "bar"; want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("handler with requires no more than two returns", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		bus := newBus()

		bus.Register(func(ctx context.Context, cmd int) (int, error, error) {
			return 0, nil, nil
		})
	})

	t.Run("handler with requires last return of two to be an error", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		bus := newBus()

		bus.Register(func(ctx context.Context, cmd int) (int, int) {
			return 0, 0
		})
	})

	t.Run("handler requires two parameters (underspecified)", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		bus := newBus()

		bus.Register(func(ctx context.Context) error { return nil })
	})

	t.Run("handler requires two parameters (overspecified)", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		bus := newBus()

		bus.Register(func(ctx context.Context, cmd string, foo int) error { return nil })
	})

	t.Run("handler requires first parameter to be a context.Context", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		bus := newBus()

		bus.Register(func(i int, cmd int) error { return nil })
	})

	t.Run("handler gets context passed through", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		bus := newBus()

		bus.Register(func(ctx context.Context, cmd string) error {
			select {
			case <-ctx.Done():
				return nil

			default:
				return errors.New("want context to be cancelled")
			}
		})

		cancel()

		if _, err := bus.Dispatch(ctx, "foo"); err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
	})

	t.Run("duplicate handlers", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		bus := newBus()

		bus.Register(func(ctx context.Context, cmd int) int { return 0 })
		bus.Register(func(ctx context.Context, cmd int) int { return 0 })
	})

	t.Run("pointers and value should be considered unique", func(t *testing.T) {
		defer func() {
			if recover() != nil {
				t.Error("want <nil>; got panic")
			}
		}()

		bus := newBus()

		bus.Register(func(ctx context.Context, cmd int) int { return 0 })
		bus.Register(func(ctx context.Context, cmd *int) int { return 0 })
		bus.Register(func(ctx context.Context, cmd **int) int { return 0 })
		bus.Register(func(ctx context.Context, cmd ***int) int { return 0 })
		bus.Register(func(ctx context.Context, cmd ****int) int { return 0 })
	})
}
