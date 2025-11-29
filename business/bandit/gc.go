package bandit

import (
	"sort"
	"time"
)

const maxArmsPerSlot = 2000

func capArms(state *LinUCBState) {
	if state == nil {
		return
	}
	if len(state.Arms) <= maxArmsPerSlot {
		return
	}

	type armInfo struct {
		productID   uint64
		lastUpdated time.Time
		count       int
	}

	infos := make([]armInfo, 0, len(state.Arms))
	for pid, arm := range state.Arms {
		infos = append(infos, armInfo{
			productID:   pid,
			lastUpdated: arm.LastUpdated,
			count:       arm.Count,
		})
	}

	// Sort ascending: oldest & least-used first
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].lastUpdated.Equal(infos[j].lastUpdated) {
			return infos[i].count < infos[j].count
		}
		return infos[i].lastUpdated.Before(infos[j].lastUpdated)
	})

	// Number of arms to drop
	toDrop := len(state.Arms) - maxArmsPerSlot
	if toDrop <= 0 {
		return
	}

	for i := 0; i < toDrop && i < len(infos); i++ {
		delete(state.Arms, infos[i].productID)
	}
}
