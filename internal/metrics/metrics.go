package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// JobsSubmitted
// JobsCompleted
// JobsFailed
// JobProcessingDuration
// QueueDepth
// WorkerErrors
// APiErrors

// Register(...)

// registry := prometheus.NewRegistry()

// | Metric                            | Prometheus type | Labels                      | Meaning                                |
// | --------------------------------- | --------------- | --------------------------- | -------------------------------------- |
// | `jobs_submitted_total`            | Counter         | `job_type`                  | Jobs successfully submitted            |
// | `jobs_completed_total`            | Counter         | `job_type`                  | Jobs successfully completed            |
// | `jobs_failed_total`               | Counter         | `job_type`, `error_type`    | Jobs that ultimately failed            |
// | `job_processing_duration_seconds` | Histogram       | `job_type`                  | Time spent processing a job            |
// | `queue_depth`                     | Gauge           | `job_type`                  | Jobs currently waiting to be processed |
// | `worker_errors_total`             | Counter         | `error_type`                | Worker-level errors                    |
// | `api_errors_total`                | Counter         | `method`, `route`, `status` | API requests resulting in errors       |

// Create metrics
type Metrics struct {
	JobsSubmitted         *prometheus.CounterVec
	JobsCompleted         *prometheus.CounterVec
	JobsFailed            *prometheus.CounterVec
	JobProcessingDuration *prometheus.HistogramVec
	QueueDepth            *prometheus.GaugeVec
	WorkerErrors          *prometheus.CounterVec
	APIErrors             *prometheus.CounterVec
}

// NewMetrics creates a new Metrics instance with initialized Prometheus metrics.
func NewMetrics(reg *prometheus.Registry) *Metrics {
	m := &Metrics{
		JobsSubmitted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "jobs_submitted_total",
				Help: "Total number of jobs successfully submitted",
			},
			[]string{"job_type"},
		),
		JobsCompleted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "jobs_completed_total",
				Help: "Total number of jobs successfully completed",
			},
			[]string{"job_type"},
		),
		JobsFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "jobs_failed_total",
				Help: "Total number of jobs that ultimately failed",
			},
			[]string{"job_type", "error_type"},
		),
		JobProcessingDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "job_processing_duration_seconds",
				Help:    "Time spent processing a job",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"job_type"},
		),
		QueueDepth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_depth",
				Help: "Number of jobs currently waiting to be processed",
			},
			[]string{"job_type"},
		),
		WorkerErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "worker_errors_total",
				Help: "Total number of worker-level errors",
			},
			[]string{"error_type"},
		),
		APIErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_errors_total",
				Help: "Total number of API requests resulting in errors",
			},
			[]string{"method", "route", "status"},
		),
	}

	reg.MustRegister(
		m.JobsSubmitted,
		m.JobsCompleted,
		m.JobsFailed,
		m.JobProcessingDuration,
		m.QueueDepth,
		m.WorkerErrors,
		m.APIErrors,
	)

	return m
}

func Handler(reg *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

func Middleware(m *Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		if recorder.statusCode >= 500 {
			m.APIErrors.WithLabelValues(
				r.Method,
				r.URL.Path,
				strconv.Itoa(recorder.statusCode),
			).Inc()
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
