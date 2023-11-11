package admin

import (
	"database/sql"
	"expvar"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/human"
)

func systemMetricsRoutes(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/metrics", func(mux *router.ServeMux) {
		mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.System.CanViewMetrics() }))

		mux.Get("/", systemMetricsGet(h), "system.metrics")
	})
}

func varAs[T any](v expvar.Var) T {
	var val T

	switch v := v.(type) {
	case *expvar.Int:
		val, _ = any(v.Value()).(T)

	case *expvar.String:
		val, _ = any(v.Value()).(T)

	case *expvar.Map:
		val, _ = any(v).(T)

	case expvar.Func:
		val, _ = v.Value().(T)
	}

	return val
}

func systemMetricsGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version := expvar.Get("version").(*expvar.Map)

		revision := varAs[string](version.Get("revision"))
		tags := varAs[string](version.Get("tags"))
		_go := varAs[string](version.Get("go"))
		os := varAs[string](version.Get("os"))
		arch := varAs[string](version.Get("arch"))
		race := varAs[string](version.Get("race"))

		uptime := varAs[time.Duration](expvar.Get("uptime"))
		now := varAs[time.Time](expvar.Get("now"))
		cgoCalls := varAs[int64](expvar.Get("cgoCalls"))
		cpus := varAs[int](expvar.Get("cpus"))
		goroutines := varAs[int](expvar.Get("goroutines"))

		memstats := varAs[runtime.MemStats](expvar.Get("memstats"))
		sqlite := varAs[sql.DBStats](h.Metrics.Get("sqlite"))

		type RequestMetrics struct {
			Label                      string
			TotalRequestsReceived      int64
			TotalRequestsInFlight      int64
			TotalResponsesSent         int64
			TotalBytesRead             int64
			TotalBytesWritten          int64
			TotalTimeUntilFirstWrite   string
			TotalTimeInHandlers        string
			TotalTimeWriting           string
			AverageBytesRead           int64
			AverageBytesWritten        int64
			AverageTimeUntilFirstWrite string
			AverageTimeInHandlers      string
			AverageTimeWriting         string
			TotalResponseStatusCodes   map[int]int64
		}

		var requests []RequestMetrics
		for _, label := range []string{"Site", "PWA", "API"} {
			key := strings.ToLower(label)

			totalRequestsReceived := varAs[int64](h.Metrics.Get(key + ".totalRequestsReceived"))
			totalResponsesSent := varAs[int64](h.Metrics.Get(key + ".totalResponsesSent"))
			totalBytesRead := varAs[int64](h.Metrics.Get(key + ".totalBytesRead"))
			totalBytesWritten := varAs[int64](h.Metrics.Get(key + ".totalBytesWritten"))
			totalTimeUntilFirstWrite := varAs[int64](h.Metrics.Get(key + ".totalTimeUntilFirstWrite"))
			totalTimeInHandlers := varAs[int64](h.Metrics.Get(key + ".totalTimeInHandlers"))
			totalTimeWriting := totalTimeInHandlers - totalTimeUntilFirstWrite
			totalResponseStatusCodesVar := varAs[*expvar.Map](h.Metrics.Get(key + ".totalResponseStatusCodes"))

			averageBytesRead := totalBytesRead / max(1, totalRequestsReceived)
			averageBytesWritten := totalBytesWritten / max(1, totalResponsesSent)
			averageTimeUntilFirstWrite := totalTimeUntilFirstWrite / max(1, totalResponsesSent)
			averageTimeInHandlers := totalTimeInHandlers / max(1, totalResponsesSent)
			averageTimeWriting := totalTimeWriting / max(1, totalResponsesSent)

			var totalResponseStatusCodes map[int]int64
			if totalResponseStatusCodesVar != nil {
				totalResponseStatusCodes = make(map[int]int64)

				totalResponseStatusCodesVar.Do(func(kv expvar.KeyValue) {
					key, _ := strconv.Atoi(kv.Key)

					totalResponseStatusCodes[key] = varAs[int64](kv.Value)
				})
			}

			requests = append(requests, RequestMetrics{
				Label:                      label,
				TotalRequestsReceived:      totalRequestsReceived,
				TotalRequestsInFlight:      totalRequestsReceived - totalResponsesSent,
				TotalResponsesSent:         totalResponsesSent,
				TotalBytesRead:             totalBytesRead,
				TotalBytesWritten:          totalBytesWritten,
				TotalTimeUntilFirstWrite:   human.DurationPrecise(time.Duration(totalTimeUntilFirstWrite)),
				TotalTimeInHandlers:        human.DurationPrecise(time.Duration(totalTimeInHandlers)),
				TotalTimeWriting:           human.DurationPrecise(time.Duration(totalTimeWriting)),
				AverageBytesRead:           averageBytesRead,
				AverageBytesWritten:        averageBytesWritten,
				AverageTimeUntilFirstWrite: human.DurationPrecise(time.Duration(averageTimeUntilFirstWrite)),
				AverageTimeInHandlers:      human.DurationPrecise(time.Duration(averageTimeInHandlers)),
				AverageTimeWriting:         human.DurationPrecise(time.Duration(averageTimeWriting)),
				TotalResponseStatusCodes:   totalResponseStatusCodes,
			})
		}

		h.HTML.View(w, r, http.StatusOK, "site/admin/system_metrics", handler.Vars{
			"Revision":          revision,
			"Tags":              tags,
			"Go":                _go,
			"OS":                os,
			"Arch":              arch,
			"Race":              race,
			"Uptime":            human.Duration(uptime),
			"Now":               now,
			"CgoCalls":          cgoCalls,
			"CPUs":              cpus,
			"Goroutines":        goroutines,
			"TotalMemOS":        memstats.Sys,
			"TotalHeapAlloc":    memstats.TotalAlloc,
			"HeapAlloc":         memstats.HeapAlloc,
			"HeapObjects":       memstats.HeapObjects,
			"GCCycles":          memstats.NumGC,
			"GCTargetHeapAlloc": memstats.NextGC,
			"LastGC":            time.Unix(0, int64(memstats.LastGC)),
			"TotalGCPause":      human.DurationPrecise(time.Duration(memstats.PauseTotalNs)),
			"SQLite": map[string]any{
				"MaxOpenConns":        sqlite.MaxOpenConnections,
				"InUseConns":          sqlite.InUse,
				"IdleConns":           sqlite.Idle,
				"ConnWaitCount":       sqlite.WaitCount,
				"TotalConnWaitTime":   human.DurationPrecise(sqlite.WaitDuration),
				"AverageConnWaitTime": human.DurationPrecise(sqlite.WaitDuration / time.Duration(max(1, sqlite.WaitCount))),
			},
			"Requests": requests,
		})
	}
}
