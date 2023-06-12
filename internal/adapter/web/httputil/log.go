package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/realip"
)

var TrustedProxies []string

func init() {
	logger.AddSkipRule("/httputil/log.go", logger.SkipFile)
}

func LogError(r *http.Request, err error) {
	LogInfof(r, "%v", err)
}

func LogInfof(r *http.Request, format string, a ...any) {
	remoteAddr, _err := realip.FromRequest(r, TrustedProxies...)
	if _err != nil {
		remoteAddr = r.RemoteAddr

		logger.Error.Println(_err)
	}

	text := fmt.Sprintf(format, a...)
	td := getTraceData(r.Context())
	request := fmt.Sprintf("%v %v", r.Method, r.URL)

	if logger.OutputStyle == logger.JSON {
		info := map[string]any{
			"traceId":    td.id,
			"userId":     td.userID,
			"request":    request,
			"remoteAddr": remoteAddr,
			"info":       text,
		}

		b, err := json.Marshal(info)
		if err != nil {
			b = []byte(err.Error())
		}

		logger.Info.Print(string(b))
	} else {
		info := fmt.Sprintf("%v (trace: %v; addr: %v; user: %v)\n", request, td.id, remoteAddr, td.userID)

		logger.Info.Printf("%v%v", info, text)
	}
}
