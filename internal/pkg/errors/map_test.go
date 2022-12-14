package errors_test

import (
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

func TestMap(t *testing.T) {
	var errs errors.Map
	if errs != nil {
		t.Errorf("want <nil>; got %v", errs)
	}

	key := "foo"
	errs.Set(key, errors.New("test error value"))
	if errs == nil {
		t.Error("want non-nil map; got <nil>")
	}

	if _, ok := errs[key]; !ok {
		t.Errorf("want key %q to be set in error map", key)
	}
}
