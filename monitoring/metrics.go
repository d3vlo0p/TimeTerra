package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	TimeterraActionExecutionSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "timeterra_action_execution_seconds",
			Help:    "Time taken to execute a Timeterra action.",
			Buckets: []float64{0.1, 0.2, 0.3, 0.5, 1, 2, 5, 10, 30, 60, 120, 180, 300},
		},
		[]string{"schedule", "action", "resource", "status"},
	)

	TimeterraScheduledCronJobs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "timeterra_scheduled_cronjobs",
			Help: "Number of Timeterra scheduled cronjobs.",
		},
		[]string{"schedule", "action"},
	)
)

// RegisterMetrics will register metrics with the global prometheus registry
func RegisterMetrics() {
	metrics.Registry.MustRegister(TimeterraActionExecutionSeconds)
	metrics.Registry.MustRegister(TimeterraScheduledCronJobs)
}
