package errsx_test

import (
	"errors"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

func TestMap(t *testing.T) {
	var errs errsx.Map
	if errs != nil {
		t.Errorf("want <nil>; got %v", errs)
	}

	key := "foo"
	testErr := errors.New("test error value")

	errs.Set(key, testErr)
	if errs == nil {
		t.Error("want non-nil map; got <nil>")
	}

	if _, ok := errs[key]; !ok {
		t.Errorf("want key %q to be set in error map", key)
	}

	if want, got := testErr.Error(), errs.Get(key); want != got {
		t.Errorf("want %q; got %q", want, got)
	}
	if want, got := "", errs.Get("does not exist"); want != got {
		t.Errorf("want %q; got %q", want, got)
	}
}
