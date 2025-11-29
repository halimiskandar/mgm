package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// Latency of the bandit Recommend HTTP handler
	BanditRecommendLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "bandit_recommend_latency_seconds",
		Help:    "Latency of bandit recommendations handler",
		Buckets: prometheus.DefBuckets,
	})

	// Total number of bandit recommendations served
	BanditRecommendRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bandit_recommend_requests_total",
		Help: "Total number of bandit recommend requests",
	})
)

func Init() {
	prometheus.MustRegister(
		BanditRecommendLatency,
		BanditRecommendRequests,
	)
}
