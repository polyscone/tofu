package testx

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/event"
)

type EventLog struct {
	got  []any
	want []any
}

func NewEventLog(broker event.Broker) *EventLog {
	var log EventLog

	broker.Clear()
	broker.ListenAny(func(ctx context.Context, data any, createdAt time.Time) {
		log.push(data)
	})

	return &log
}

func (e *EventLog) push(data any) {
	e.got = append(e.got, data)
}

func (e *EventLog) Expect(data any) {
	e.want = append(e.want, data)
}

func (e *EventLog) Check(t *testing.T) bool {
	t.Helper()

	if want, got := len(e.want), len(e.got); want != got {
		var events string
		for _, data := range e.got {
			events += fmt.Sprintf("  %#v\n", data)
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

func CheckEvents(t *testing.T, wantEventData, gotEventData []event.Event) bool {
	t.Helper()

	if want, got := len(wantEventData), len(gotEventData); want != got {
		var events string
		for _, data := range gotEventData {
			events += fmt.Sprintf("  %#v\n", data)
		}

		events = strings.TrimSuffix(events, "\n")

		t.Errorf("\nwant %v events; got %v:\n%v", want, got, events)

		return false
	}

	for i, want := range wantEventData {
		got := gotEventData[i]

		if want != got {
			t.Errorf("\nfor event %v:\nwant %#v\ngot  %#v", i, want, got)

			return false
		}
	}

	return true
}
