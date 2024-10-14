package testx

import (
	"fmt"
	"strings"
	"testing"

	"github.com/polyscone/tofu/internal/event"
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
		var events string
		for _, evt := range e.got {
			events += fmt.Sprintf("  %#v\n", evt)
		}

		events = strings.TrimSuffix(events, "\n")

		t.Errorf("\nwant %v events; got %v:\n%v", want, got, events)

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
		var events string
		for _, evt := range gotEvents {
			events += fmt.Sprintf("  %#v\n", evt)
		}

		events = strings.TrimSuffix(events, "\n")

		t.Errorf("\nwant %v events; got %v:\n%v", want, got, events)

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
