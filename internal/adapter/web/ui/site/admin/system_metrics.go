package admin

import (
	"database/sql"
	"expvar"
	"math"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/http/router"
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

		type DatabaseMetrics struct {
			Label                       string
			MaxOpenConns                int
			InUseConns                  int
			IdleConns                   int
			ConnWaitCount               int64
			TotalConnWaitTime           time.Duration
			AverageConnWaitTime         time.Duration
			TotalTransactionsBegun      int64
			TotalTransactionsCommitted  int64
			TotalTransactionsRolledBack int64
			TransactionCommitRate       float64
			TransactionRollbackRate     float64
			TotalQueriesExecuted        int64
			TotalQueryTime              time.Duration
			AverageQueryTime            time.Duration
			AverageTransactionQueries   int64
		}

		type RequestMetrics struct {
			Label                      string
			TotalRequestsReceived      int64
			TotalRequestsInFlight      int64
			TotalResponsesSent         int64
			TotalConnectionsHijacked   int64
			TotalBytesRead             uint64
			TotalBytesWritten          uint64
			TotalTimeUntilFirstWrite   time.Duration
			TotalTimeInHandlers        time.Duration
			TotalTimeWriting           time.Duration
			AverageBytesRead           uint64
			AverageBytesWritten        uint64
			AverageTimeUntilFirstWrite time.Duration
			AverageTimeInHandlers      time.Duration
			AverageTimeWriting         time.Duration
			TotalResponseStatusCodes   map[int]int64
		}

		var databases []DatabaseMetrics
		var requests []RequestMetrics
		h.Metrics.Do(func(kv expvar.KeyValue) {
			switch {
			case strings.HasPrefix(kv.Key, "database."):
				database := varAs[*expvar.Map](kv.Value)
				if database == nil {
					return
				}

				_, label, _ := strings.Cut(kv.Key, "database.")

				stats := varAs[sql.DBStats](database.Get("stats"))

				totalTransactionsBegun := varAs[int64](database.Get("totalTransactionsBegun"))
				totalTransactionsCommitted := varAs[int64](database.Get("totalTransactionsCommitted"))
				totalTransactionsRolledBack := varAs[int64](database.Get("totalTransactionsRolledBack"))
				totalQueriesExecuted := varAs[int64](database.Get("totalQueriesExecuted"))
				totalQueryTime := varAs[int64](database.Get("totalQueryTime"))

				averageConnWaitTime := stats.WaitDuration / time.Duration(max(1, stats.WaitCount))
				transactionCommitRate := float64(totalTransactionsCommitted) / float64(totalTransactionsBegun) * 100
				transactionRollbackRate := 100 - transactionCommitRate
				averageQueryTime := totalQueryTime / max(1, totalQueriesExecuted)
				averageTransactionQueries := int64(math.Round(float64(totalQueriesExecuted) / max(1, float64(totalTransactionsBegun))))

				databases = append(databases, DatabaseMetrics{
					Label:                       label,
					MaxOpenConns:                stats.MaxOpenConnections,
					InUseConns:                  stats.InUse,
					IdleConns:                   stats.Idle,
					ConnWaitCount:               stats.WaitCount,
					TotalConnWaitTime:           stats.WaitDuration,
					AverageConnWaitTime:         averageConnWaitTime,
					TotalTransactionsBegun:      totalTransactionsBegun,
					TotalTransactionsCommitted:  totalTransactionsCommitted,
					TotalTransactionsRolledBack: totalTransactionsRolledBack,
					TransactionCommitRate:       transactionCommitRate,
					TransactionRollbackRate:     transactionRollbackRate,
					TotalQueriesExecuted:        totalQueriesExecuted,
					TotalQueryTime:              time.Duration(totalQueryTime),
					AverageQueryTime:            time.Duration(averageQueryTime),
					AverageTransactionQueries:   averageTransactionQueries,
				})

			case strings.HasPrefix(kv.Key, "requests."):
				group := varAs[*expvar.Map](kv.Value)
				if group == nil {
					return
				}

				_, label, _ := strings.Cut(kv.Key, "requests.")

				totalRequestsReceived := varAs[int64](group.Get("totalRequestsReceived"))
				totalResponsesSent := varAs[int64](group.Get("totalResponsesSent"))
				totalConnectionsHijacked := varAs[int64](group.Get("totalConnectionsHijacked"))
				totalBytesRead := int(varAs[int64](group.Get("totalBytesRead")))
				totalBytesWritten := int(varAs[int64](group.Get("totalBytesWritten")))
				totalTimeUntilFirstWrite := varAs[int64](group.Get("totalTimeUntilFirstWrite"))
				totalTimeInHandlers := varAs[int64](group.Get("totalTimeInHandlers"))
				totalResponseStatusCodesVar := varAs[*expvar.Map](group.Get("totalResponseStatusCodes"))

				totalRequestsInFlight := totalRequestsReceived - totalResponsesSent
				totalTimeWriting := totalTimeInHandlers - totalTimeUntilFirstWrite
				averageBytesRead := uint64(math.Round(float64(totalBytesRead) / max(1, float64(totalRequestsReceived))))
				averageBytesWritten := uint64(math.Round(float64(totalBytesWritten) / max(1, float64(totalResponsesSent))))
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
					TotalRequestsInFlight:      totalRequestsInFlight,
					TotalResponsesSent:         totalResponsesSent,
					TotalConnectionsHijacked:   totalConnectionsHijacked,
					TotalBytesRead:             uint64(totalBytesRead),
					TotalBytesWritten:          uint64(totalBytesWritten),
					TotalTimeUntilFirstWrite:   time.Duration(totalTimeUntilFirstWrite),
					TotalTimeInHandlers:        time.Duration(totalTimeInHandlers),
					TotalTimeWriting:           time.Duration(totalTimeWriting),
					AverageBytesRead:           uint64(averageBytesRead),
					AverageBytesWritten:        uint64(averageBytesWritten),
					AverageTimeUntilFirstWrite: time.Duration(averageTimeUntilFirstWrite),
					AverageTimeInHandlers:      time.Duration(averageTimeInHandlers),
					AverageTimeWriting:         time.Duration(averageTimeWriting),
					TotalResponseStatusCodes:   totalResponseStatusCodes,
				})
			}
		})

		h.HTML.View(w, r, http.StatusOK, "site/admin/system_metrics", handler.Vars{
			"Revision":          revision,
			"Tags":              tags,
			"Go":                _go,
			"OS":                os,
			"Arch":              arch,
			"Race":              race,
			"Uptime":            uptime,
			"Now":               now,
			"CgoCalls":          cgoCalls,
			"CPUs":              cpus,
			"Goroutines":        goroutines,
			"ReservedMemOS":     memstats.Sys,
			"TotalHeapAlloc":    memstats.TotalAlloc,
			"HeapAlloc":         memstats.HeapAlloc,
			"HeapObjects":       memstats.HeapObjects,
			"GCCycles":          memstats.NumGC,
			"GCTargetHeapAlloc": memstats.NextGC,
			"LastGC":            time.Unix(0, int64(memstats.LastGC)),
			"TotalGCPause":      time.Duration(memstats.PauseTotalNs),
			"Databases":         databases,
			"Requests":          requests,
		})
	}
}
