package notifications

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	NotificationProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "notification_processed_total",
		Help: "Successfully processed notification events",
	}, []string{"event_type"})
	NotificationFailed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "notification_failed_total",
		Help: "Failed notification processing attempts",
	}, []string{"event_type"})
	EmailSent = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "email_sent_total",
		Help: "Successfully sent notification emails",
	}, []string{"event_type"})
	EmailFailed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "email_failed_total",
		Help: "Failed notification email attempts",
	}, []string{"event_type"})
	OutboxPending = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_pending",
		Help: "Pending and currently processing outbox events",
	})
	registerMetricsOnce sync.Once
)

func RegisterMetrics() {
	registerMetricsOnce.Do(func() {
		prometheus.MustRegister(NotificationProcessed, NotificationFailed, EmailSent, EmailFailed, OutboxPending)
	})
}
