package csrf_test

import (
	"context"
	"errors"
	"testing"

	"github.com/polyscone/tofu/pkg/csrf"
)

func TestCSRF(t *testing.T) {
	t.Run("set a basic token", func(t *testing.T) {
		ctx := context.Background()
		ctx, err := csrf.SetToken(ctx, nil)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := true, csrf.IsNew(ctx); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		token := csrf.MaskedToken(ctx)
		if err := csrf.Check(ctx, token); err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		ctx, err = csrf.SetToken(ctx, token)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if err := csrf.Check(ctx, token); err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
	})

	t.Run("masked token is unique every call", func(t *testing.T) {
		ctx := context.Background()
		ctx, err := csrf.SetToken(ctx, nil)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		token1 := csrf.MaskedToken(ctx)
		token2 := csrf.MaskedToken(ctx)
		token3 := csrf.MaskedToken(ctx)

		if want, got := string(token1), string(token2); want == got {
			t.Error("want unique strings; got equal")
		}
		if want, got := string(token2), string(token3); want == got {
			t.Error("want unique strings; got equal")
		}
	})

	t.Run("error when token is not correct length", func(t *testing.T) {
		ctx := context.Background()
		ctx, err := csrf.SetToken(ctx, nil)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		token := csrf.MaskedToken(ctx)

		if _, err := csrf.SetToken(ctx, token[:len(token)-1]); err == nil {
			t.Error("want error; got <nil>")
		}
	})

	t.Run("generate a unique token when empty", func(t *testing.T) {
		ctx := context.Background()
		ctx, err := csrf.SetToken(ctx, nil)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		token1 := csrf.MaskedToken(ctx)

		if want, got := "", string(token1); want == got {
			t.Errorf("want unique strings; got equal")
		}

		if err := csrf.Check(ctx, token1); err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		ctx, err = csrf.SetToken(ctx, nil)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		if err := csrf.Check(ctx, token1); !errors.Is(err, csrf.ErrInvalidToken) {
			t.Errorf("want csrf.ErrInvalidToken; got %q", err)
		}

		token2 := csrf.MaskedToken(ctx)
		if err := csrf.Check(ctx, token2); err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
	})

	t.Run("renew token", func(t *testing.T) {
		ctx := context.Background()
		ctx, err := csrf.SetToken(ctx, nil)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := true, csrf.IsNew(ctx); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		token1 := csrf.MaskedToken(ctx)
		if err := csrf.Check(ctx, token1); err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		ctx, err = csrf.SetToken(ctx, token1)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := false, csrf.IsNew(ctx); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if err := csrf.RenewToken(ctx); err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := true, csrf.IsNew(ctx); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		if err := csrf.Check(ctx, token1); !errors.Is(err, csrf.ErrInvalidToken) {
			t.Errorf("want csrf.ErrInvalidToken; got %q", err)
		}

		token2 := csrf.MaskedToken(ctx)
		if err := csrf.Check(ctx, token2); err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
	})

	t.Run("check token", func(t *testing.T) {
		ctx := context.Background()
		ctx, err := csrf.SetToken(ctx, nil)
		if err != nil {
			t.Fatalf("want <nil>; got %q", err)
		}

		t.Run("same token", func(t *testing.T) {
			token := csrf.MaskedToken(ctx)
			err := csrf.Check(ctx, token)
			if err != nil {
				t.Errorf("want <nil>; got %q", err)
			}
		})

		t.Run("different token", func(t *testing.T) {
			ctx2 := context.Background()
			ctx2, err := csrf.SetToken(ctx2, nil)
			if err != nil {
				t.Errorf("want <nil>; got %q", err)
			}

			token := csrf.MaskedToken(ctx2)
			if err := csrf.Check(ctx, token); !errors.Is(err, csrf.ErrInvalidToken) {
				t.Errorf("want csrf.ErrInvalidToken; got %q", err)
			}
		})

		t.Run("empty token", func(t *testing.T) {
			if err := csrf.Check(ctx, nil); !errors.Is(err, csrf.ErrEmptyToken) {
				t.Errorf("want csrf.ErrEmptyToken; got %q", err)
			}
		})
	})
}
