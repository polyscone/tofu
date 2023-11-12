package middleware

import (
	"bufio"
	"expvar"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func Metrics(metrics *expvar.Map, group string) Middleware {
	statusCodesKey := group + ".totalResponseStatusCodes"
	statusCodesMap, ok := metrics.Get(statusCodesKey).(*expvar.Map)
	if !ok {
		statusCodesMap = &expvar.Map{}

		metrics.Set(statusCodesKey, statusCodesMap)
	}

	suffixes := []string{
		"totalRequestsReceived",
		"totalResponsesSent",
		"totalBytesRead",
		"totalBytesWritten",
		"totalTimeUntilFirstWrite",
		"totalTimeInHandlers",
	}
	for _, suffix := range suffixes {
		if metrics.Get(group+"."+suffix) == nil {
			metrics.Set(group+"."+suffix, &expvar.Int{})
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &metricsResponseWriter{
				ResponseWriter: w,
				rc:             http.NewResponseController(w),
				group:          group,
				metrics:        metrics,
			}

			if r.Body != nil {
				r.Body = &metricsRequestReader{
					ReadCloser: r.Body,
					group:      group,
					metrics:    metrics,
				}
			}

			metrics.Add(group+".totalRequestsReceived", 1)

			next(rw, r)

			if rw.statusCode == 0 {
				rw.statusCode = http.StatusOK
			}

			// The end time is used to estimate the amount of time spent
			// inside of HTTP handler functions
			//
			// If the connection was hijacked then we just assume the end
			// of the handler was at the time of the hijacking, since
			// the handler function could still be running in an infinite
			// loop whilst the TCP connection is being used for other purposes
			//
			// If there was no first write time recorded then we assume it's
			// the hijacked time, if there is one, or the current time
			end := time.Now()
			if !rw.hijackedAt.IsZero() {
				end = rw.hijackedAt
			}
			if rw.firstWriteAt.IsZero() {
				rw.firstWriteAt = end
			}

			statusCodesMap.Add(strconv.Itoa(rw.statusCode), 1)

			metrics.Add(group+".totalResponsesSent", 1)
			metrics.Add(group+".totalTimeUntilFirstWrite", rw.firstWriteAt.Sub(start).Nanoseconds())
			metrics.Add(group+".totalTimeInHandlers", end.Sub(start).Nanoseconds())
		}
	}
}

type metricsRequestReader struct {
	io.ReadCloser
	group   string
	metrics *expvar.Map
}

func (r *metricsRequestReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)

	r.metrics.Add(r.group+".totalBytesRead", int64(n))

	return n, err
}

var _ Unwrapper = (*metricsResponseWriter)(nil)

type metricsResponseWriter struct {
	http.ResponseWriter
	mu           sync.Mutex
	rc           *http.ResponseController
	group        string
	metrics      *expvar.Map
	statusCode   int
	firstWriteAt time.Time
	hijackedAt   time.Time
}

func (w *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *metricsResponseWriter) FlushError() error {
	w.recordFirstWrite()

	return w.rc.Flush()
}

func (w *metricsResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.recordHijack()

	return w.rc.Hijack()
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	w.recordFirstWrite()

	w.metrics.Add(w.group+".totalBytesWritten", int64(len(b)))

	return w.ResponseWriter.Write(b)
}

func (w *metricsResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)

	w.recordFirstWrite()

	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
}

func (w *metricsResponseWriter) recordFirstWrite() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.firstWriteAt.IsZero() {
		w.firstWriteAt = time.Now()
	}
}

func (w *metricsResponseWriter) recordHijack() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.hijackedAt.IsZero() {
		w.hijackedAt = time.Now()
	}

	// If there is no first write time recorded yet then we just
	// set it to be the same as the hijacked time
	//
	// This is because after the TCP connection has been hijacked
	// it can still be written to, but we're only interedted in
	// recording writes in the context of a normal HTTP handler here
	if w.firstWriteAt.IsZero() {
		w.firstWriteAt = w.hijackedAt
	}
}
