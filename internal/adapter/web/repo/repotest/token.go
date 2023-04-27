package repotest

import (
	"context"
	"encoding/base32"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/token"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/repo"
	"github.com/polyscone/tofu/internal/pkg/testutil/quick"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

func RunTokenTests(t *testing.T, tokens token.Repo) {
	t.Run("sequence", func(t *testing.T) {
		ctx := context.Background()
		email1 := text.GenerateEmail()
		email2 := text.GenerateEmail()
		email3 := text.GenerateEmail()

		// Generate a token for an email
		tok1, err := tokens.AddActivationToken(ctx, email1, 1*time.Minute)
		if err != nil {
			t.Fatal(err)
		}

		// We expect generated tokens to be returned base32 encoded
		decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(tok1)
		if err != nil {
			t.Fatal(err)
		}

		// The decoded token must be at least 128 bits in length
		if want, got := 16, len(decoded); want > got {
			t.Fatalf("want at least %v bytes of entropy; got %v", want, got)
		}

		// Generating another token for the same email should succeed and
		// replace the old token so it can't be used in place of the new one
		// This means that trying to find the old token should result in
		// a not found error
		tok1Old := tok1
		tok1, err = tokens.AddActivationToken(ctx, email1, 1*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		_, err = tokens.FindActivationTokenEmail(ctx, tok1Old)
		if want, got := repo.ErrNotFound, err; !errors.Is(got, want) {
			t.Errorf("want repo.ErrNotFound; got %q", got)
		}

		// Generating another token for a different email should result in
		// another unique token
		tok2, err := tokens.AddActivationToken(ctx, email2, 1*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if tok1 == tok2 {
			t.Error("want unique tokens; got duplicates")
		}

		// Querying a token successfully should return the associated email
		emailCmp, err := tokens.FindActivationTokenEmail(ctx, tok1)
		if err != nil {
			t.Fatal(err)
		}
		if want, got := email1, emailCmp; want != got {
			t.Errorf("want email %q; got %q", want, got)
		}

		// Trying to consume a token and then finding its data should result in
		// not found because the token should have been deleted
		err = tokens.ConsumeActivationToken(ctx, tok1)
		if err != nil {
			t.Fatal(err)
		}
		_, err = tokens.FindActivationTokenEmail(ctx, tok1)
		if want, got := repo.ErrNotFound, err; !errors.Is(got, want) {
			t.Errorf("want repo.ErrNotFound; got %q", got)
		}

		// Trying to consume or find an expired token should fail with not found
		tok3, err := tokens.AddActivationToken(ctx, email3, -1*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		_, err = tokens.FindActivationTokenEmail(ctx, tok3)
		if want, got := repo.ErrNotFound, err; !errors.Is(got, want) {
			t.Errorf("want repo.ErrNotFound; got %q", got)
		}
		err = tokens.ConsumeActivationToken(ctx, tok3)
		if want, got := repo.ErrNotFound, err; !errors.Is(got, want) {
			t.Errorf("want repo.ErrNotFound; got %q", got)
		}
	})

	t.Run("token kinds", func(t *testing.T) {
		ctx := context.Background()

		quick.Check(t, func(email text.Email) bool {
			tok, err := tokens.AddActivationToken(ctx, email, 1*time.Minute)
			if err != nil {
				return false
			}

			val, err := tokens.FindActivationTokenEmail(ctx, tok)
			if err != nil {
				return false
			}

			err = tokens.ConsumeActivationToken(ctx, tok)
			if err != nil {
				return false
			}

			return email == val
		})

		quick.Check(t, func(email text.Email) bool {
			tok, err := tokens.AddResetPasswordToken(ctx, email, 1*time.Minute)
			if err != nil {
				return false
			}

			val, err := tokens.FindResetPasswordTokenEmail(ctx, tok)
			if err != nil {
				return false
			}

			err = tokens.ConsumeResetPasswordToken(ctx, tok)
			if err != nil {
				return false
			}

			return email == val
		})
	})
}
