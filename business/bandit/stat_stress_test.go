//go:build !integration

package bandit

import (
	"fmt"
	"math/rand"
	"testing"
)

// minimal fake LinUCBState just for counting
type fakeArm struct {
	count int
}

type fakeState struct {
	Arms map[uint64]*fakeArm
}

func newFakeState() *fakeState {
	return &fakeState{Arms: make(map[uint64]*fakeArm)}
}

// scenario params
const (
	stressNumUsers        = 20000
	stressNumSegments     = 3
	stressNumProducts     = 500
	stressFeedbackPerUser = 50
)

var stressSlots = []string{"home_top", "pdp_similar"}

func TestStateGrowth_GlobalVsGlobalPlusUser(t *testing.T) {

	// --- 1) GLOBAL ONLY (current behavior) ---

	globalStates := make(map[string]*fakeState)

	for u := 1; u <= stressNumUsers; u++ {
		userID := uint(u)
		for _, slot := range stressSlots {
			seg := int(userID) % stressNumSegments
			key := fmt.Sprintf("%s|seg=%d", slot, seg)

			st, ok := globalStates[key]
			if !ok {
				st = newFakeState()
				globalStates[key] = st
			}

			seenProd := make(map[uint64]struct{})
			for i := 0; i < stressFeedbackPerUser; i++ {
				pid := uint64(rand.Intn(stressNumProducts))
				seenProd[pid] = struct{}{}
				arm, ok := st.Arms[pid]
				if !ok {
					arm = &fakeArm{}
					st.Arms[pid] = arm
				}
				arm.count++
			}

			_ = seenProd
		}
	}

	totalGlobalStates := len(globalStates)
	totalGlobalArms := 0
	for _, st := range globalStates {
		totalGlobalArms += len(st.Arms)
	}

	t.Logf("[GLOBAL ONLY] states=%d arms=%d", totalGlobalStates, totalGlobalArms)

	// --- 2) GLOBAL + USER-LEVEL ---

	globalStates2 := make(map[string]*fakeState)
	userStates := make(map[string]*fakeState)

	for u := 1; u <= stressNumUsers; u++ {
		userID := uint(u)
		for _, slot := range stressSlots {
			seg := int(userID) % stressNumSegments

			gKey := fmt.Sprintf("%s|seg=%d|global", slot, seg)
			uKey := fmt.Sprintf("%s|seg=%d|user=%d", slot, seg, userID)

			gSt, ok := globalStates2[gKey]
			if !ok {
				gSt = newFakeState()
				globalStates2[gKey] = gSt
			}
			uSt, ok := userStates[uKey]
			if !ok {
				uSt = newFakeState()
				userStates[uKey] = uSt
			}

			seenProd := make(map[uint64]struct{})
			for i := 0; i < stressFeedbackPerUser; i++ {
				pid := uint64(rand.Intn(stressNumProducts))
				seenProd[pid] = struct{}{}

				// global update
				gArm, ok := gSt.Arms[pid]
				if !ok {
					gArm = &fakeArm{}
					gSt.Arms[pid] = gArm
				}
				gArm.count++

				// user-level update
				uArm, ok := uSt.Arms[pid]
				if !ok {
					uArm = &fakeArm{}
					uSt.Arms[pid] = uArm
				}
				uArm.count++
			}

			_ = seenProd
		}
	}

	totalGlobalStates2 := len(globalStates2)
	totalGlobalArms2 := 0
	for _, st := range globalStates2 {
		totalGlobalArms2 += len(st.Arms)
	}

	totalUserStates := len(userStates)
	totalUserArms := 0
	for _, st := range userStates {
		totalUserArms += len(st.Arms)
	}

	t.Logf("[GLOBAL+USER] globalStates=%d globalArms=%d userStates=%d userArms=%d",
		totalGlobalStates2, totalGlobalArms2, totalUserStates, totalUserArms)
}
