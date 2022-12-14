package testutil

import (
	"testing"

	"github.com/polyscone/tofu/internal/pkg/event"
)

func CheckEvents(t *testing.T, wantEvents, gotEvents []event.Event) bool {
	t.Helper()

	if want, got := len(wantEvents), len(gotEvents); want != got {
		t.Errorf("want %v events; got %v", want, got)

		return false
	}

	for i, want := range wantEvents {
		got := gotEvents[i]

		if want != got {
			t.Errorf("\nfor event %v:\nwant %#v\ngot  %#v", i, want, got)

			return false
		}
	}

	return true
}
