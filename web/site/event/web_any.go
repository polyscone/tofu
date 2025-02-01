package event

import (
	"context"
	"encoding/json"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/event"
	"github.com/polyscone/tofu/web/site/ui"
)

func WebAnyHandler(h *ui.Handler) event.AnyHandler {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		panic("site: web any handler: failed to read build info")
	}

	return func(ctx context.Context, data any, createdAt time.Time) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		go func() {
			b, err := json.Marshal(data)
			if err == nil {
				const kind = "site"
				name := eventName(bi.Main.Path+"/", reflect.TypeOf(data))

				ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()

				if err := h.Repo.Web.LogDomainEvent(ctx, kind, name, string(b)); err != nil {
					logger.Error("web any: add domain event", "error", err)
				}
			}
		}()
	}
}

func eventName(pkgTrimPrefix string, typ reflect.Type) string {
	var prefix string
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		prefix += "*"
	}

	pkg := strings.TrimPrefix(typ.PkgPath(), pkgTrimPrefix)

	return prefix + pkg + "." + typ.Name()
}
