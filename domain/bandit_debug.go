package domain

type DebugRecommendation struct {
	ProductID         uint64  `json:"product_id"`
	OfflineScore      float64 `json:"offline_score"`      // from mock_recommendations.score
	OfflineNormalized float64 `json:"offline_normalized"` // 0–1
	BanditMean        float64 `json:"bandit_mean"`        // θᵀx
	BanditUncertainty float64 `json:"bandit_uncertainty"` // sqrt(xᵀA⁻¹x)
	BanditUCB         float64 `json:"bandit_ucb"`         // mean + α·uncertainty
	FinalScore        float64 `json:"final_score"`        // wBandit*UCB + wOffline*offline_norm

	Features []float64      `json:"features,omitempty"` // raw feature vector
	Context  map[string]any `json:"context,omitempty"`  // time_bucket, dow, platform, dll
	Segment  int            `json:"segment"`            // which segment used
	Variant  int            `json:"variant"`            // which variant used
}
