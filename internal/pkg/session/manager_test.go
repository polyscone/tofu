package session_test

import (
	"context"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/session"
)

func TestSessionManager(t *testing.T) {
	t.Run("initial session setup", func(t *testing.T) {
		sm := session.NewManager(session.NewMemoryRepo())
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
		sm := session.NewManager(session.NewMemoryRepo())
		ctx := context.Background()

		ctx, err := sm.Load(ctx, "")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		sm.Set(ctx, "key", true)

		if want, got := true, sm.GetBool(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := true, sm.PopBool(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := false, sm.PopBool(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		sm.Set(ctx, "key", 123)

		if want, got := 123, sm.GetInt(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 123, sm.PopInt(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := 0, sm.PopInt(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		sm.Set(ctx, "key", float32(123.45))

		if want, got := float32(123.45), sm.GetFloat32(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := float32(123.45), sm.PopFloat32(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := float32(0.0), sm.PopFloat32(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		sm.Set(ctx, "key", float64(123.45))

		if want, got := float64(123.45), sm.GetFloat64(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := float64(123.45), sm.PopFloat64(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := float64(0.0), sm.PopFloat64(ctx, "key"); want != got {
			t.Errorf("want %v; got %v", want, got)
		}

		sm.Set(ctx, "key", "Hello, World!")

		if want, got := "Hello, World!", sm.GetString(ctx, "key"); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
		if want, got := "Hello, World!", sm.PopString(ctx, "key"); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
		if want, got := "", sm.PopString(ctx, "key"); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("membership tests", func(t *testing.T) {
		sm := session.NewManager(session.NewMemoryRepo())
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
		sm := session.NewManager(session.NewMemoryRepo())
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

	t.Run("renew a session id", func(t *testing.T) {
		sm := session.NewManager(session.NewMemoryRepo())
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
		sm := session.NewManager(session.NewMemoryRepo())
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
}
