package event

import (
	"github.com/polyscone/tofu/internal/adapter/web/event"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
)

func SignedInWithPasswordHandler(h *ui.Handler) any {
	return event.SignedInWithPasswordHandler(h.Handler)
}
