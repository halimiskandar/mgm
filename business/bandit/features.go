// business/bandit/features.go
package bandit

import (
	"fmt"
	"time"
)

func stateSlotKey(slot string, segment int) string {
	return fmt.Sprintf("%s|seg=%d", slot, segment)
}

// time-of-day → 0, 0.33, 0.66, 1
func timeBucket(hour int) float64 {
	switch {
	case hour < 6:
		return 0.0
	case hour < 12:
		return 0.33
	case hour < 18:
		return 0.66
	default:
		return 1.0
	}
}

// day-of-week: 0..6 → 0..1
func dowBucket(d int) float64 {
	if d < 0 || d > 6 {
		return 0.0
	}
	return float64(d) / 6.0
}

// slot string hash → [0,1]
func slotHash(slot string) float64 {
	var h uint32 = 2166136261
	for i := 0; i < len(slot); i++ {
		h ^= uint32(slot[i])
		h *= 16777619
	}
	return float64(h%1000) / 1000.0
}

// product hash → [0,1]
func productHash(productID uint64) float64 {
	return float64(productID%1000) / 1000.0
}

// user hash → [0,1]
func userHash(userID uint) float64 {
	return float64(userID%1000) / 1000.0
}

func buildFeatureVector(userID uint, slot string, productID uint64, cfg Config, seg int) [linUCBFeatureDim]float64 {
	now := time.Now()
	hour := now.Hour()
	dow := int(now.Weekday())

	out := [linUCBFeatureDim]float64{}

	if cfg.Features.UseBias {
		out[0] = 1.0
	}
	if cfg.Features.UseTimeBucket {
		out[1] = timeBucket(hour)
	}
	if cfg.Features.UseDowBucket {
		out[2] = dowBucket(dow)
	}
	if cfg.Features.UseSlotHash {
		out[3] = slotHash(slot)
	}
	if cfg.Features.UseSegment && cfg.NumSegments > 0 {
		out[4] = float64(seg) / float64(cfg.NumSegments)
	}

	// index 5: either product hash or mixed user+product hash
	if cfg.Features.UseUserHash {
		// simple mixture of user and product identity
		out[5] = 0.5*productHash(productID) + 0.5*userHash(userID)
	} else if cfg.Features.UseProductHash {
		out[5] = productHash(productID)
	}

	return out
}
