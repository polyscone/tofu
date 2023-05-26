package testutil

import (
	"testing"

	"github.com/polyscone/tofu/internal/pkg/event"
)

type EventLog struct {
	got  []event.Event
	want []event.Event
}

func NewEventLog(broker event.Broker) *EventLog {
	var log EventLog

	broker.Clear()
	broker.ListenAny(func(evt event.Event) {
		log.push(evt)
	})

	return &log
}

func (e *EventLog) push(evt event.Event) {
	e.got = append(e.got, evt)
}

func (e *EventLog) Expect(evt event.Event) {
	e.want = append(e.want, evt)
}

func (e *EventLog) Check(t *testing.T) bool {
	t.Helper()

	if want, got := len(e.want), len(e.got); want != got {
		t.Errorf("want %v events; got %v", want, got)

		return false
	}

	for i, want := range e.want {
		got := e.got[i]

		if want != got {
			t.Errorf("\nfor event %v:\nwant %#v\ngot  %#v", i, want, got)

			return false
		}
	}

	return true
}

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
