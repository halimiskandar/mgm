package bandit

import (
	"sort"
	"time"
)

func capArms(state *LinUCBState, maxArms int) {
	if state == nil {
		return
	}
	if maxArms <= 0 {
		// 0 or negative: treat as "no cap"
		return
	}
	if len(state.Arms) <= maxArms {
		return
	}

	type kv struct {
		productID uint64
		updated   time.Time
	}

	list := make([]kv, 0, len(state.Arms))
	for pid, arm := range state.Arms {
		updated := arm.LastUpdated
		if updated.IsZero() {
			updated = time.Time{} // treat zero as the oldest
		}
		list = append(list, kv{
			productID: pid,
			updated:   updated,
		})
	}

	// newest first
	sort.Slice(list, func(i, j int) bool {
		return list[i].updated.After(list[j].updated)
	})

	// keep [0:maxArms), delete the rest
	for i := maxArms; i < len(list); i++ {
		delete(state.Arms, list[i].productID)
	}
}
