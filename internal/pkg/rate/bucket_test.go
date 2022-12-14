package rate_test

import (
	"sync"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/rate"
)

func TestBucket(t *testing.T) {
	t.Run("correctly leak and replenish tokens", func(t *testing.T) {
		now := time.Now().UTC()
		capacity, replenish := 50, 1
		bucket := rate.NewTokenBucket(capacity, replenish)

		if want, got := 50, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 45, errors.Must(bucket.Leak(5, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 45, errors.Must(bucket.Leak(-5, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(2 * time.Second)

		if want, got := 47, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 40, errors.Must(bucket.Leak(7, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(3 * time.Second)

		if want, got := 43, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(20 * time.Second)

		if want, got := 50, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 20, errors.Must(bucket.Leak(30, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(-20 * time.Second)

		if want, got := 20, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(30 * time.Second)

		if want, got := 30, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		location := errors.Must(time.LoadLocation("America/Los_Angeles"))
		now = now.In(location)

		if want, got := 30, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(1 * time.Second)

		if want, got := 31, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.UTC()
		now = now.Add(1 * time.Second)

		if want, got := 32, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(500 * time.Millisecond)

		if want, got := 32, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(500 * time.Millisecond)

		if want, got := 33, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		n, err := bucket.Leak(133, now)
		if want, got := rate.ErrInsufficientTokens, err; !errors.Is(got, want) {
			t.Errorf("want rate.ErrInsufficientTokens; got %q", got)
		}
		if want, got := 33, n; want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 0, errors.Must(bucket.Leak(33, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(1 * time.Second)

		if want, got := 1, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 0, errors.Must(bucket.Leak(1, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		_, err = bucket.Leak(1, now)
		if want, got := rate.ErrInsufficientTokens, err; !errors.Is(got, want) {
			t.Errorf("want rate.ErrInsufficientTokens; got %q", got)
		}

		now = now.Add(5 * time.Second)

		if want, got := 5, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 2, errors.Must(bucket.Leak(3, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		_, err = bucket.Leak(3, now)
		if want, got := rate.ErrInsufficientTokens, err; !errors.Is(got, want) {
			t.Errorf("want rate.ErrInsufficientTokens; got %q", got)
		}
	})

	t.Run("replenish more than one token", func(t *testing.T) {
		now := time.Now().UTC()
		capacity, replenish := 50, 3
		bucket := rate.NewTokenBucket(capacity, replenish)

		if want, got := 50, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 40, errors.Must(bucket.Leak(10, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(1 * time.Second)

		if want, got := 43, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
	})

	t.Run("safe concurrent usage of leak and replenish", func(t *testing.T) {
		full := 1000
		half, double := full/2, full*2
		now := time.Now().UTC()
		capacity, replenish := full, 1
		bucket := rate.NewTokenBucket(capacity, replenish)

		// We expect some inconsistencies with timing at the limits of the
		// bucket capacity, so to avoid those inconsistencies for the test
		// we start by leaking half the tokens
		if want, got := half, errors.Must(bucket.Leak(half, now)); want != got {
			t.Fatalf("want %v; got %v", want, got)
		}

		var wg sync.WaitGroup

		wg.Add(double)
		for i := 0; i < double; i++ {
			now = now.Add(1 * time.Second)

			go func(now time.Time) {
				bucket.Leak(1, now)

				wg.Done()
			}(now)
		}

		wg.Wait()

		// Since we leaked half the tokens at the beginning to avoid
		// inconsistencies in test results around the capacity limits, we
		// expect half a bucket of tokens after consistently leaking and
		// replenishing one token
		if want, got := half, errors.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
	})
}
