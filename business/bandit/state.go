package bandit

const linUCBFeatureDim = 4

// Per arm/product LinUCB parameters.
type LinUCBArmState struct {
	A     [linUCBFeatureDim][linUCBFeatureDim]float64 `json:"A"`
	B     [linUCBFeatureDim]float64                   `json:"b"`
	Count int                                         `json:"count"`
}

// Overall state for a slot.
type LinUCBState struct {
	Alpha float64                    `json:"alpha"`
	Arms  map[uint64]*LinUCBArmState `json:"arms"` // key: productID
}

// Create a default state for a new slot.
func newDefaultState() *LinUCBState {
	return &LinUCBState{
		Alpha: 1.0,
		Arms:  make(map[uint64]*LinUCBArmState),
	}
}

// Create a new arm with A initialized to identity.
func newArmState() *LinUCBArmState {
	var a [linUCBFeatureDim][linUCBFeatureDim]float64
	for i := 0; i < linUCBFeatureDim; i++ {
		a[i][i] = 1.0
	}
	return &LinUCBArmState{
		A: a,
		B: [linUCBFeatureDim]float64{},
	}
}
