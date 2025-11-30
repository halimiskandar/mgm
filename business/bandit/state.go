package bandit

import "time"

const linUCBFeatureDim = 7

// Per arm/product LinUCB parameters.
type LinUCBArmState struct {
	A           [linUCBFeatureDim][linUCBFeatureDim]float64 `json:"A"`
	B           [linUCBFeatureDim]float64                   `json:"b"`
	Count       int                                         `json:"count"`
	LastUpdated time.Time                                   `json:"last_updated"`
}

// Overall state for a slot.
type LinUCBState struct {
	Alpha float64                    `json:"alpha"`
	Arms  map[uint64]*LinUCBArmState `json:"arms"` // key: productID
}

// Create a new arm with A initialized to identity.
func newArmState() *LinUCBArmState {
	var A [linUCBFeatureDim][linUCBFeatureDim]float64
	for i := 0; i < linUCBFeatureDim; i++ {
		A[i][i] = 0.1
	}
	return &LinUCBArmState{
		A:           A,
		B:           [linUCBFeatureDim]float64{},
		Count:       0,
		LastUpdated: time.Now(),
	}
}

// Create a default state for a new slot.
func newDefaultState() *LinUCBState {
	return &LinUCBState{
		Alpha: 1.0,
		Arms:  make(map[uint64]*LinUCBArmState),
	}
}
