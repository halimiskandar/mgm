package domain

type DebugRecommendation struct {
	ProductID         uint64  `json:"product_id"`
	OfflineScore      float64 `json:"offline_score"`      // from mock_recommendations.score
	OfflineNormalized float64 `json:"offline_normalized"` // 0–1
	BanditMean        float64 `json:"bandit_mean"`        // θᵀx
	BanditUncertainty float64 `json:"bandit_uncertainty"` // sqrt(xᵀA⁻¹x)
	BanditUCB         float64 `json:"bandit_ucb"`         // mean + α·uncertainty
	FinalScore        float64 `json:"final_score"`        // wBandit*UCB + wOffline*offline_norm
}
