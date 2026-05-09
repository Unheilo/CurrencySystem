package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Metrics struct {
	Registry *prometheus.Registry

	// RED (gRPC-level)
	ReqCount    *prometheus.CounterVec
	ReqDuration *prometheus.HistogramVec

	// ECB client
	ECBRequestDuration *prometheus.HistogramVec
	ECBErrors          *prometheus.CounterVec

	// Worker
	FetchJobDuration *prometheus.HistogramVec
	FetchJobRuns     *prometheus.CounterVec

	// DB
	DBQueryDuration *prometheus.HistogramVec

	// Cache
	CacheHits   prometheus.Counter
	CacheMisses prometheus.Counter
}

func New() *Metrics {
	m := &Metrics{
		Registry: prometheus.NewRegistry(),

		ReqCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "currency_request_count_total"},
			[]string{"method", "code"},
		),
		ReqDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "currency_ecb_api_request_duration_seconds",
				Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"outcome"},
		),
		ECBErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "currency_ecb_api_errors_total"},
			[]string{"reason"},
		),

		FetchJobDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "currency_fetch_job_duration_seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
			},
			[]string{"trigger"}, // "startup" / "cron"
		),
		FetchJobRuns: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "currency_fetch_job_runs_total"},
			[]string{"operation"},
		),

		CacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "currency_cache_hits_total",
		}),
		CacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "currency_cache_misses_total",
		}),
	}

	m.Registry.MustRegister(
		m.ReqCount, m.ReqDuration,
		m.ECBRequestDuration, m.ECBErrors,
		m.FetchJobDuration, m.FetchJobRuns,
		m.DBQueryDuration,
		m.CacheHits, m.CacheMisses,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return m
}
