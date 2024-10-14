package middleware

import (
	"bufio"
	"expvar"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

func Metrics(metrics *expvar.Map, name string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var statusCodes *expvar.Map
			group, ok := metrics.Get(name).(*expvar.Map)
			if ok {
				statusCodes, _ = group.Get("totalResponseStatusCodes").(*expvar.Map)
			} else {
				statusCodes = &expvar.Map{}
				group = &expvar.Map{}

				group.Set("totalResponseStatusCodes", statusCodes)
				group.Set("totalRequestsReceived", &expvar.Int{})
				group.Set("totalResponsesSent", &expvar.Int{})
				group.Set("totalConnectionsHijacked", &expvar.Int{})
				group.Set("totalBytesRead", &expvar.Int{})
				group.Set("totalBytesWritten", &expvar.Int{})
				group.Set("totalTimeUntilFirstWrite", &expvar.Int{})
				group.Set("totalTimeInHandlers", &expvar.Int{})

				metrics.Set(name, group)
			}

			start := time.Now()
			rw := &metricsResponseWriter{
				ResponseWriter: w,
				rc:             http.NewResponseController(w),
				group:          group,
			}

			if r.Body != nil {
				r.Body = &metricsRequestReader{
					ReadCloser: r.Body,
					group:      group,
				}
			}

			group.Add("totalRequestsReceived", 1)

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

			statusCodes.Add(strconv.Itoa(rw.statusCode), 1)

			group.Add("totalResponsesSent", 1)
			group.Add("totalTimeUntilFirstWrite", rw.firstWriteAt.Sub(start).Nanoseconds())
			group.Add("totalTimeInHandlers", end.Sub(start).Nanoseconds())
		}
	}
}

type metricsRequestReader struct {
	io.ReadCloser
	group *expvar.Map
}

func (r *metricsRequestReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)

	r.group.Add("totalBytesRead", int64(n))

	return n, err
}

var _ Unwrapper = (*metricsResponseWriter)(nil)

type metricsResponseWriter struct {
	http.ResponseWriter
	rc           *http.ResponseController
	group        *expvar.Map
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
	conn, bufrw, err := w.rc.Hijack()
	if err == nil {
		w.recordHijack()

		w.group.Add("totalConnectionsHijacked", 1)
	}

	return conn, bufrw, err
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	w.recordFirstWrite()

	w.group.Add("totalBytesWritten", int64(len(b)))

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
	if w.firstWriteAt.IsZero() {
		w.firstWriteAt = time.Now()
	}
}

func (w *metricsResponseWriter) recordHijack() {
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
