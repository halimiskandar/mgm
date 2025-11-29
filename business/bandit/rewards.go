package bandit

import (
	"fmt"
	"myGreenMarket/domain"
)

// RewardForEvent turns a BanditEvent into a numeric reward using the current config.
func (cfg Config) RewardForEvent(ev domain.BanditEvent) (float64, error) {
	var base float64

	switch ev.EventType {
	case "impression":
		base = cfg.RewardImpression
	case "click":
		base = cfg.RewardClick
	case "atc":
		base = cfg.RewardATC
	case "order":
		base = cfg.RewardOrder
	default:
		return 0, fmt.Errorf("unknown event type: %s", ev.EventType)
	}

	// business value component (dynamic, from DB)
	if ev.Value > 0 {
		base += cfg.ValueWeight * ev.Value
	}

	return base, nil
}
