package repotest

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/app/account"
)

func AccountSignInAttemptLogs(ctx context.Context, t *testing.T, newRepo func() account.ReadWriter) {
	t.Run("find and save", func(t *testing.T) {
		repo := newRepo()

		log, err := repo.FindSignInAttemptLogByEmail(ctx, "foo@example.com")
		if err != nil {
			t.Fatal(err)
		}

		if want, got := "foo@example.com", log.Email; want != got {
			t.Errorf("want email to be %q; got %q", want, got)
		}
		if want, got := 0, log.Attempts; want != got {
			t.Errorf("want attempts to be %v; got %v", want, got)
		}
		if !log.LastAttemptAt.IsZero() {
			t.Errorf("want last attempt at to be zero; got %v", log.LastAttemptAt)
		}

		log.Attempts = 5
		log.LastAttemptAt = time.Now()
		if err := repo.SaveSignInAttemptLog(ctx, log); err != nil {
			t.Fatal(err)
		}

		log, err = repo.FindSignInAttemptLogByEmail(ctx, "foo@example.com")
		if err != nil {
			t.Fatal(err)
		}

		if want, got := "foo@example.com", log.Email; want != got {
			t.Errorf("want email to be %q; got %q", want, got)
		}
		if want, got := 5, log.Attempts; want != got {
			t.Errorf("want attempts to be %v; got %v", want, got)
		}
		if log.LastAttemptAt.IsZero() {
			t.Error("want last attempt at to be populated; got zero")
		}
	})
}
