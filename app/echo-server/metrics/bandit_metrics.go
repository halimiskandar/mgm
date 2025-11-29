package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	RecommendDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "bandit_recommend_latency_seconds",
		Help:    "Latency of bandit recommendation endpoint",
		Buckets: prometheus.DefBuckets,
	})

	RecommendTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bandit_recommend_total",
		Help: "Total recommendations served",
	})

	ExploreCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bandit_explore_events_total",
		Help: "How many times exploration noise affected ranking",
	})
)

func Init() {
	prometheus.MustRegister(RecommendDuration, RecommendTotal, ExploreCount)
}
