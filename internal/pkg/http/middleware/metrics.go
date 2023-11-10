package middleware

import (
	"expvar"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func Metrics(metrics *expvar.Map, keys ...string) Middleware {
	var statusCodeMaps []*expvar.Map
	for _, key := range keys {
		statusCodesKey := key + ".totalResponseStatusCodes"
		statusCodesMap := metrics.Get(statusCodesKey)
		if statusCodesMap == nil {
			statusCodesMap = &expvar.Map{}

			metrics.Set(statusCodesKey, statusCodesMap)
		}

		statusCodeMaps = append(statusCodeMaps, statusCodesMap.(*expvar.Map))

		metrics.Set(key+".totalRequestsReceived", &expvar.Int{})
		metrics.Set(key+".totalResponsesSent", &expvar.Int{})
		metrics.Set(key+".totalBytesWritten", &expvar.Int{})
		metrics.Set(key+".totalTimeUntilFirstWrite", &expvar.Int{})
		metrics.Set(key+".totalTimeInHandlers", &expvar.Int{})
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &metricsResponseWriter{
				ResponseWriter: w,
				rc:             http.NewResponseController(w),
				keys:           keys,
				metrics:        metrics,
			}

			for _, key := range keys {
				metrics.Add(key+".totalRequestsReceived", 1)
			}

			next(rw, r)

			if rw.statusCode == 0 {
				rw.statusCode = http.StatusOK
			}

			end := time.Now()
			if rw.end.IsZero() {
				rw.end = end
			}

			for i, key := range keys {
				statusCodeMaps[i].Add(strconv.Itoa(rw.statusCode), 1)

				metrics.Add(key+".totalResponsesSent", 1)
				metrics.Add(key+".totalTimeUntilFirstWrite", rw.end.Sub(start).Nanoseconds())
				metrics.Add(key+".totalTimeInHandlers", end.Sub(start).Nanoseconds())
			}
		}
	}
}

var _ Unwrapper = (*metricsResponseWriter)(nil)

type metricsResponseWriter struct {
	http.ResponseWriter
	mu         sync.Mutex
	rc         *http.ResponseController
	keys       []string
	metrics    *expvar.Map
	statusCode int
	end        time.Time
}

func (w *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *metricsResponseWriter) FlushError() error {
	w.recordEnd()

	return w.rc.Flush()
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	w.recordEnd()

	for _, key := range w.keys {
		w.metrics.Add(key+".totalBytesWritten", int64(len(b)))
	}

	return w.ResponseWriter.Write(b)
}

func (w *metricsResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)

	w.recordEnd()

	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
}

func (w *metricsResponseWriter) recordEnd() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.end.IsZero() {
		w.end = time.Now()
	}
}
