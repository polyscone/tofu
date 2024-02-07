package rate_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/rate"
)

func TestBucket(t *testing.T) {
	t.Run("correctly leak and replenish tokens", func(t *testing.T) {
		now := time.Now().UTC()
		capacity, replenish := 50.0, 1.0
		bucket := rate.NewTokenBucket(capacity, replenish)

		if want, got := 50, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 45, errsx.Must(bucket.Leak(5, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 45, errsx.Must(bucket.Leak(-5, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(2 * time.Second)

		if want, got := 47, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 40, errsx.Must(bucket.Leak(7, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(3 * time.Second)

		if want, got := 43, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(20 * time.Second)

		if want, got := 50, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 20, errsx.Must(bucket.Leak(30, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(-20 * time.Second)

		if want, got := 20, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(30 * time.Second)

		if want, got := 30, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		location := errsx.Must(time.LoadLocation("America/Los_Angeles"))
		now = now.In(location)

		if want, got := 30, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(1 * time.Second)

		if want, got := 31, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.UTC()
		now = now.Add(1 * time.Second)

		if want, got := 32, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(500 * time.Millisecond)

		if want, got := 32, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(500 * time.Millisecond)

		if want, got := 33, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		n, err := bucket.Leak(133, now)
		if want, got := rate.ErrInsufficientTokens, err; !errors.Is(got, want) {
			t.Errorf("want rate.ErrInsufficientTokens; got %q", got)
		}
		if want, got := 33, n; want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 0, errsx.Must(bucket.Leak(33, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(1 * time.Second)

		if want, got := 1, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 0, errsx.Must(bucket.Leak(1, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		_, err = bucket.Leak(1, now)
		if want, got := rate.ErrInsufficientTokens, err; !errors.Is(got, want) {
			t.Errorf("want rate.ErrInsufficientTokens; got %q", got)
		}

		now = now.Add(5 * time.Second)

		if want, got := 5, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 2, errsx.Must(bucket.Leak(3, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		_, err = bucket.Leak(3, now)
		if want, got := rate.ErrInsufficientTokens, err; !errors.Is(got, want) {
			t.Errorf("want rate.ErrInsufficientTokens; got %q", got)
		}
	})

	t.Run("replenish more than one token", func(t *testing.T) {
		now := time.Now().UTC()
		capacity, replenish := 50.0, 3.0
		bucket := rate.NewTokenBucket(capacity, replenish)

		if want, got := 50, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 40, errsx.Must(bucket.Leak(10, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(1 * time.Second)

		if want, got := 43, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
	})

	t.Run("consume and replenish fractional tokens", func(t *testing.T) {
		now := time.Now().UTC()
		capacity, replenish := 50.0, 0.5
		bucket := rate.NewTokenBucket(capacity, replenish)

		if want, got := 50, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 42, errsx.Must(bucket.Leak(7.7, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(1 * time.Second)

		if want, got := 42, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(1 * time.Second)

		if want, got := 43, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(1 * time.Second)

		if want, got := 43, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		now = now.Add(1 * time.Second)

		if want, got := 44, errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
	})

	t.Run("safe concurrent usage of leak and replenish", func(t *testing.T) {
		full := 1000.0
		half, double := full/2, full*2
		now := time.Now().UTC()
		capacity, replenish := full, 1.0
		bucket := rate.NewTokenBucket(capacity, replenish)

		// We expect some inconsistencies with timing at the limits of the
		// bucket capacity, so to avoid those inconsistencies for the test
		// we start by leaking half the tokens
		if want, got := int(half), errsx.Must(bucket.Leak(half, now)); want != got {
			t.Fatalf("want %v; got %v", want, got)
		}

		var wg sync.WaitGroup

		wg.Add(int(double))
		for range int(double) {
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
		if want, got := int(half), errsx.Must(bucket.Leak(0, now)); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
	})
}
