package httputil

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/realip"
	"golang.org/x/exp/slog"
)

var TrustedProxies []string

func Log(logger *slog.Logger, level slog.Level, r *http.Request, msg string, args ...any) {
	remoteAddr, _err := realip.FromRequest(r, TrustedProxies...)
	if _err != nil {
		remoteAddr = r.RemoteAddr

		logger.Error("realip from request", "error", _err)
	}

	td := getTraceData(r.Context())

	args = append(args, "id", td.id)
	args = append(args, "method", r.Method)
	args = append(args, "remoteAddr", remoteAddr)
	args = append(args, "url", r.URL.String())
	args = append(args, "user", td.userID)

	logger.Log(nil, level, msg, args...)
}

func LogDebug(logger *slog.Logger, r *http.Request, msg string, args ...any) {
	Log(logger, slog.LevelDebug, r, msg, args...)
}

func LogInfo(logger *slog.Logger, r *http.Request, msg string, args ...any) {
	Log(logger, slog.LevelInfo, r, msg, args...)
}

func LogWarn(logger *slog.Logger, r *http.Request, msg string, args ...any) {
	Log(logger, slog.LevelWarn, r, msg, args...)
}

func LogError(logger *slog.Logger, r *http.Request, msg string, args ...any) {
	Log(logger, slog.LevelError, r, msg, args...)
}
