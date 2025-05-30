package session

import (
	"context"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/errsx"
)

func TestManager(t *testing.T, newRepo func() ReadWriter) {
	t.Run("initial session setup", func(t *testing.T) {
		sm := errsx.Must(NewManager(newRepo()))
		ctx := context.Background()

		initID := "qux"
		ctx, err := sm.Load(ctx, initID)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := "", sm.GetString(ctx, "foo"); want != got {
			t.Errorf("want %q; got %q", want, got)
		}

		sm.Set(ctx, "foo", "bar")

		if want, got := "bar", sm.GetString(ctx, "foo"); want != got {
			t.Errorf("want %q; got %q", want, got)
		}

		id, err := sm.Commit(ctx)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if got, cmp := id, ""; got == cmp {
			t.Errorf("want different strings; got equal (%q)", got)
		}
		if got, cmp := id, initID; got == cmp {
			t.Errorf("want different strings; got equal (%q)", got)
		}
	})

	t.Run("get, set, and pop", func(t *testing.T) {
		sm := errsx.Must(NewManager(newRepo()))
		ctx := context.Background()

		ctx = errsx.Must(sm.Load(ctx, ""))
		id := errsx.Must(sm.Commit(ctx))

		t.Run("bool", func(t *testing.T) {
			ctx = errsx.Must(sm.Load(ctx, id))

			sm.Set(ctx, "key", true)

			errsx.Must(sm.Commit(ctx))
			ctx = errsx.Must(sm.Load(ctx, id))

			if want, got := true, sm.GetBool(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := true, sm.PopBool(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := false, sm.PopBool(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}

			errsx.Must(sm.Commit(ctx))
		})

		t.Run("int", func(t *testing.T) {
			ctx = errsx.Must(sm.Load(ctx, id))

			sm.Set(ctx, "key", 123)

			errsx.Must(sm.Commit(ctx))
			ctx = errsx.Must(sm.Load(ctx, id))

			if want, got := 123, sm.GetInt(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := 123, sm.PopInt(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := 0, sm.PopInt(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}

			errsx.Must(sm.Commit(ctx))
		})

		t.Run("float32", func(t *testing.T) {
			ctx = errsx.Must(sm.Load(ctx, id))

			sm.Set(ctx, "key", float32(123.45))

			errsx.Must(sm.Commit(ctx))
			ctx = errsx.Must(sm.Load(ctx, id))

			if want, got := float32(123.45), sm.GetFloat32(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := float32(123.45), sm.PopFloat32(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := float32(0.0), sm.PopFloat32(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}

			errsx.Must(sm.Commit(ctx))
		})

		t.Run("float64", func(t *testing.T) {
			ctx = errsx.Must(sm.Load(ctx, id))

			sm.Set(ctx, "key", float64(123.45))

			errsx.Must(sm.Commit(ctx))
			ctx = errsx.Must(sm.Load(ctx, id))

			if want, got := float64(123.45), sm.GetFloat64(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := float64(123.45), sm.PopFloat64(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := float64(0.0), sm.PopFloat64(ctx, "key"); want != got {
				t.Errorf("want %v; got %v", want, got)
			}

			errsx.Must(sm.Commit(ctx))
		})

		t.Run("string", func(t *testing.T) {
			ctx = errsx.Must(sm.Load(ctx, id))

			sm.Set(ctx, "key", "Hello, World!")

			errsx.Must(sm.Commit(ctx))
			ctx = errsx.Must(sm.Load(ctx, id))

			if want, got := "Hello, World!", sm.GetString(ctx, "key"); want != got {
				t.Errorf("want %q; got %q", want, got)
			}
			if want, got := "Hello, World!", sm.PopString(ctx, "key"); want != got {
				t.Errorf("want %q; got %q", want, got)
			}
			if want, got := "", sm.PopString(ctx, "key"); want != got {
				t.Errorf("want %q; got %q", want, got)
			}

			errsx.Must(sm.Commit(ctx))
		})

		t.Run("strings", func(t *testing.T) {
			ctx = errsx.Must(sm.Load(ctx, id))

			sm.Set(ctx, "keyblah", []string{"Foo", "Bar", "Baz"})

			errsx.Must(sm.Commit(ctx))
			ctx = errsx.Must(sm.Load(ctx, id))

			equal := func(s1, s2 []string) bool {
				if len(s1) != len(s2) {
					return false
				}

				for i, value := range s1 {
					if value != s2[i] {
						return false
					}
				}

				return true
			}

			if want, got := []string{"Foo", "Bar", "Baz"}, sm.GetStrings(ctx, "keyblah"); !equal(want, got) {
				t.Errorf("want %q; got %q", want, got)
			}
			if want, got := []string{"Foo", "Bar", "Baz"}, sm.PopStrings(ctx, "keyblah"); !equal(want, got) {
				t.Errorf("want %q; got %q", want, got)
			}
			if want, got := []string(nil), sm.PopStrings(ctx, "keyblah"); !equal(want, got) {
				t.Errorf("want %q; got %q", want, got)
			}

			errsx.Must(sm.Commit(ctx))
		})

		t.Run("time.Time", func(t *testing.T) {
			ctx = errsx.Must(sm.Load(ctx, id))

			now := time.Now()

			sm.Set(ctx, "key", now)

			errsx.Must(sm.Commit(ctx))
			ctx = errsx.Must(sm.Load(ctx, id))

			if want, got := now, sm.GetTime(ctx, "key"); !want.Equal(got) {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := now, sm.PopTime(ctx, "key"); !want.Equal(got) {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := (time.Time{}), sm.PopTime(ctx, "key"); !want.Equal(got) {
				t.Errorf("want %v; got %v", want, got)
			}

			errsx.Must(sm.Commit(ctx))
		})
	})

	t.Run("membership tests", func(t *testing.T) {
		sm := errsx.Must(NewManager(newRepo()))
		ctx := context.Background()

		ctx, err := sm.Load(ctx, "")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := false, sm.Has(ctx, "foo"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		sm.Set(ctx, "foo", "bar")

		if want, got := true, sm.Has(ctx, "foo"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		sm.Delete(ctx, "foo")

		if want, got := false, sm.Has(ctx, "foo"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
	})

	t.Run("loading an existing session", func(t *testing.T) {
		sm := errsx.Must(NewManager(newRepo()))
		ctx := context.Background()

		ctx, err := sm.Load(ctx, "")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		sm.Set(ctx, "foo", "bar")

		originalID, err := sm.Commit(ctx)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		ctx, err = sm.Load(ctx, originalID)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := "bar", sm.GetString(ctx, "foo"); want != got {
			t.Errorf("want %q; got %q", want, got)
		}

		id, err := sm.Commit(ctx)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := originalID, id; want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("clear session data", func(t *testing.T) {
		sm := errsx.Must(NewManager(newRepo()))
		ctx := context.Background()

		ctx = errsx.Must(sm.Load(ctx, ""))
		id := errsx.Must(sm.Commit(ctx))

		ctx = errsx.Must(sm.Load(ctx, id))

		sm.Set(ctx, "foo", true)
		sm.Set(ctx, "bar", true)

		errsx.Must(sm.Commit(ctx))
		ctx = errsx.Must(sm.Load(ctx, id))

		if key := "foo"; !sm.Has(ctx, key) {
			t.Errorf("want key %v to exist", key)
		}
		if key := "bar"; !sm.Has(ctx, key) {
			t.Errorf("want key %v to exist", key)
		}

		sm.Clear(ctx)

		errsx.Must(sm.Commit(ctx))
		ctx = errsx.Must(sm.Load(ctx, id))

		if key := "foo"; sm.Has(ctx, key) {
			t.Errorf("want key %v to not exist", key)
		}
		if key := "bar"; sm.Has(ctx, key) {
			t.Errorf("want key %v to not exist", key)
		}

		errsx.Must(sm.Commit(ctx))
	})

	t.Run("renew a session id", func(t *testing.T) {
		sm := errsx.Must(NewManager(newRepo()))
		ctx := context.Background()

		ctx, err := sm.Load(ctx, "")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		originalID, err := sm.Commit(ctx)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		if err := sm.Renew(ctx); err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		id, err := sm.Commit(ctx)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if got, cmp := originalID, id; got == cmp {
			t.Errorf("want different strings; got equal (%q)", got)
		}
	})

	t.Run("destroying an existing session", func(t *testing.T) {
		sm := errsx.Must(NewManager(newRepo()))
		ctx := context.Background()

		ctx, err := sm.Load(ctx, "")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		originalID, err := sm.Commit(ctx)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		ctx, err = sm.Load(ctx, originalID)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		sm.Destroy(ctx)

		id, err := sm.Commit(ctx)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := originalID, id; want != got {
			t.Errorf("want %q; got %q", want, got)
		}

		ctx, err = sm.Load(ctx, id)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		id, err = sm.Commit(ctx)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if got, cmp := originalID, id; got == cmp {
			t.Errorf("want different strings; got equal (%q)", got)
		}
	})

	t.Run("renew all existing session", func(t *testing.T) {
		sm := errsx.Must(NewManager(newRepo()))
		ctx := context.Background()

		ctx, err := sm.Load(ctx, "")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		originalID, err := sm.Commit(ctx)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		ctx, err = sm.Load(ctx, originalID)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		sm.RenewKey()

		id, err := sm.Commit(ctx)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if want, got := originalID, id; want != got {
			t.Errorf("want %q; got %q", want, got)
		}

		ctx, err = sm.Load(ctx, id)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		id, err = sm.Commit(ctx)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}
		if got, cmp := originalID, id; got == cmp {
			t.Errorf("want different strings; got equal (%q)", got)
		}
	})
}
