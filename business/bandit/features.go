package bandit

import (
	"fmt"
	"hash/fnv"
	"time"
)

// global key
func stateGlobalKey(slot string, segment int) string {
	return fmt.Sprintf("%s|seg=%d|global", slot, segment)
}

// user state: personal delta for a specific user
func stateUserKey(slot string, segment int, userID uint) string {
	return fmt.Sprintf("%s|seg=%d|user=%d", slot, segment, userID)
}

func timeBucketFromHour(hour int) float64 {
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
func computeTimeBucket(t time.Time) string {
	h := t.Hour()
	switch {
	case h < 6:
		return "night"
	case h < 12:
		return "morning"
	case h < 18:
		return "afternoon"
	default:
		return "evening"
	}
}

// timeBucketFromLabel maps the labels produced by computeTimeBucket
// ("night", "morning", "afternoon", "evening") to the same numeric buckets.
func timeBucketFromLabel(label string) (float64, bool) {
	switch label {
	case "night":
		return 0.0, true
	case "morning":
		return 0.33, true
	case "afternoon":
		return 0.66, true
	case "evening":
		return 1.0, true
	default:
		// unknown – let caller decide to fallback
		return 0.5, false
	}
}

// dowBucket encodes day-of-week (0=Sunday .. 6=Saturday) into [0, 1].
func dowBucket(dow int) float64 {
	if dow < 0 {
		dow = 0
	} else if dow > 6 {
		dow = 6
	}
	return float64(dow) / 6.0
}

// hashToUnit deterministically hashes a string into [0, 1].
func hashToUnit(s string) float64 {
	if s == "" {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return float64(h.Sum32()) / float64(^uint32(0))
}

func slotHash(slot string) float64 {
	if slot == "" {
		return 0
	}
	return hashToUnit("slot:" + slot)
}

func platformBucket(platform string) float64 {
	if platform == "" {
		// neutral default when platform is unknown
		return 0.5
	}
	return hashToUnit("platform:" + platform)
}

func userHash(userID uint) float64 {
	if userID == 0 {
		return 0
	}
	return hashToUnit(fmt.Sprintf("user:%d", userID))
}

func productHash(productID uint64) float64 {
	if productID == 0 {
		return 0
	}
	return hashToUnit(fmt.Sprintf("product:%d", productID))
}

func buildFeatureVector(
	userID uint,
	slot string,
	productID uint64,
	cfg Config,
	seg int,
	ctxMap map[string]any,
) [linUCBFeatureDim]float64 {

	// Defaults from "now"; can be overridden by ctxMap.
	now := time.Now()
	hour := now.Hour()
	dow := int(now.Weekday())
	platform := ""

	var tbFromLabel float64
	hasTbLabel := false

	if ctxMap != nil {
		if tbLabel, ok := ctxMap["time_bucket"].(string); ok {
			if v, ok2 := timeBucketFromLabel(tbLabel); ok2 {
				tbFromLabel = v
				hasTbLabel = true
			}
		}
		if d, ok := ctxMap["dow"].(int); ok {
			dow = d
		}
		if p, ok := ctxMap["platform"].(string); ok {
			platform = p
		}
	}

	var x [linUCBFeatureDim]float64

	// index 0: bias
	if cfg.Features.UseBias {
		x[0] = 1.0
	}

	// index 1: time bucket
	if cfg.Features.UseTimeBucket {
		if hasTbLabel {
			x[1] = tbFromLabel
		} else {
			x[1] = timeBucketFromHour(hour)
		}
	}

	// index 2: day-of-week bucket
	if cfg.Features.UseDowBucket {
		x[2] = dowBucket(dow)
	}

	// index 3: platform (always encoded if present)
	x[3] = platformBucket(platform)

	// index 4: slot hash
	if cfg.Features.UseSlotHash {
		x[4] = slotHash(slot)
	}

	// index 5: segment
	if cfg.Features.UseSegment && cfg.NumSegments > 0 {
		x[5] = float64(seg) / float64(cfg.NumSegments)
	}

	// index 6: user/product hash with tier & campaign seasoning
	if cfg.Features.UseUserHash {
		// Build a composite string that includes tier & campaign from context
		extra := ""
		if tier, ok := ctxMap["user_tier"].(string); ok && tier != "" {
			extra += "|tier:" + tier
		}
		if camp, ok := ctxMap["campaign_id"].(string); ok && camp != "" {
			extra += "|camp:" + camp
		}

		base := fmt.Sprintf("user:%d|prod:%d%s", userID, productID, extra)
		x[6] = hashToUnit(base) // hashToUnit should map string → [0,1] float
	} else if cfg.Features.UseProductHash {
		x[6] = productHash(productID)
	}

	return x
}
