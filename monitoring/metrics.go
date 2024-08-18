package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	TimeterraActionLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "timeterra_action_latency_seconds",
			Help:    "Time taken to execute a Timeterra action.",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.5, 1, 2, 3, 5, 10, 20, 30, 60},
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

	TimeTerraNotificationLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "timeterra_notification_latency_seconds",
			Help:    "Time taken to send a Timeterra notification.",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.5, 1, 2, 3, 5, 10, 20, 30, 60},
		},
		[]string{"type", "schedule", "action", "resource", "status"},
	)

	TimeTerraNotificationPolicies = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "timeterra_notification_policies",
			Help: "Number of Timeterra notification policies.",
		},
		[]string{"type", "schedule"},
	)
)

// RegisterMetrics will register metrics with the global prometheus registry
func RegisterMetrics() {
	metrics.Registry.MustRegister(TimeterraActionLatency)
	metrics.Registry.MustRegister(TimeterraScheduledCronJobs)
	metrics.Registry.MustRegister(TimeTerraNotificationPolicies)
}
