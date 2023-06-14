package httputil

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/realip"
	"golang.org/x/exp/slog"
)

var TrustedProxies []string

func Log(log func(string, ...any), r *http.Request, msg string, args ...any) {
	remoteAddr, _err := realip.FromRequest(r, TrustedProxies...)
	if _err != nil {
		remoteAddr = r.RemoteAddr

		slog.Error("realip from request", "error", _err)
	}

	td := getTraceData(r.Context())

	args = append(args, "id", td.id)
	args = append(args, "method", r.Method)
	args = append(args, "remoteAddr", remoteAddr)
	args = append(args, "url", r.URL.String())
	args = append(args, "user", td.userID)

	log(msg, args...)
}

func LogError(r *http.Request, msg string, args ...any) {
	Log(slog.Error, r, msg, args...)
}

func LogInfo(r *http.Request, msg string, args ...any) {
	Log(slog.Info, r, msg, args...)
}
