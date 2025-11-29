package domain

type BanditFeatureFlags struct {
	UseBias        bool `json:"use_bias"`
	UseTimeBucket  bool `json:"use_time_bucket"`
	UseDowBucket   bool `json:"use_dow_bucket"`
	UseSlotHash    bool `json:"use_slot_hash"`
	UseSegment     bool `json:"use_segment"`
	UseProductHash bool `json:"use_product_hash"`
	UseUserHash    bool `json:"use_user_hash"`
}

type BanditConfig struct {
	Slot    string `json:"slot" gorm:"column:slot"`
	Variant int    `json:"variant" gorm:"column:variant"`

	WBandit      float64 `json:"w_bandit" gorm:"column:w_bandit"`
	WOffline     float64 `json:"w_offline" gorm:"column:w_offline"`
	ExploreNoise float64 `json:"explore_noise" gorm:"column:explore_noise"`
	Alpha        float64 `json:"alpha" gorm:"column:alpha"`

	//   business value
	ValueWeight float64 `json:"value_weight" gorm:"column:value_weight"`

	//  per-event base rewards
	RewardImpression float64 `json:"reward_impression" gorm:"column:reward_impression"`
	RewardClick      float64 `json:"reward_click" gorm:"column:reward_click"`
	RewardATC        float64 `json:"reward_atc" gorm:"column:reward_atc"`
	RewardOrder      float64 `json:"reward_order" gorm:"column:reward_order"`

	NumSegments int `json:"num_segments" gorm:"column:num_segments"`
	NumVariants int `json:"num_variants" gorm:"column:num_variants"`

	FeaturesRaw []byte             `json:"-" gorm:"column:features"`
	Features    BanditFeatureFlags `json:"features" gorm:"-"`
}
