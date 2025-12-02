package bandit

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	BanditFeedbackEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bandit_feedback_events_total",
			Help: "Count of bandit feedback events by slot, event_type, segment, and variant.",
		},
		[]string{"slot", "event_type", "segment", "variant"},
	)
)

func init() {
	prometheus.MustRegister(BanditFeedbackEventsTotal)
}
