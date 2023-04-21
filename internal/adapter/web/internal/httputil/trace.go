package httputil

import (
	"context"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/internal/sesskey"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

type ctxKey byte

const ctxTraceData ctxKey = iota

type traceData struct {
	id     string
	userID string
}

var fallbackTraceData = traceData{
	id:     "n/a",
	userID: "n/a",
}

func TraceRequest(sm *session.Manager, errorHandler middleware.ErrorHandler) middleware.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			id, err := uuid.NewV4()
			if err != nil {
				if errorHandler == nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				} else {
					errorHandler(w, r, err)
				}

				return
			}

			userID := sm.GetString(ctx, sesskey.UserID)

			td := traceData{
				id:     id.String(),
				userID: userID,
			}

			ctx = context.WithValue(ctx, ctxTraceData, &td)
			r = r.WithContext(ctx)

			next(w, r)
		}
	}
}

func getTraceData(ctx context.Context) *traceData {
	value := ctx.Value(ctxTraceData)
	if value == nil {
		return &fallbackTraceData
	}

	td, ok := value.(*traceData)
	if !ok {
		return &fallbackTraceData
	}

	if td.id == "" {
		td.id = fallbackTraceData.id
	}

	if td.userID == "" {
		td.userID = fallbackTraceData.userID
	}

	return td
}
