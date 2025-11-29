// business/bandit/math.go
package bandit

import (
	"fmt"
	"math"
)

const decayRate = 0.001 // soft forgetting

// y = A * x
func matVecMul(A [linUCBFeatureDim][linUCBFeatureDim]float64, x [linUCBFeatureDim]float64) [linUCBFeatureDim]float64 {
	var y [linUCBFeatureDim]float64
	for i := range linUCBFeatureDim {
		sum := 0.0
		for j := range linUCBFeatureDim {
			sum += A[i][j] * x[j]
		}
		y[i] = sum
	}
	return y
}

func dot(a, b [linUCBFeatureDim]float64) float64 {
	sum := 0.0
	for i := range linUCBFeatureDim {
		sum += a[i] * b[i]
	}
	return sum
}

// A := A + x x^T
func addOuter(A *[linUCBFeatureDim][linUCBFeatureDim]float64, x [linUCBFeatureDim]float64) {
	for i := range linUCBFeatureDim {
		for j := range linUCBFeatureDim {
			(*A)[i][j] += x[i] * x[j]
		}
	}
}

// b := b + r x
func addScaled(b *[linUCBFeatureDim]float64, x [linUCBFeatureDim]float64, r float64) {
	for i := range linUCBFeatureDim {
		(*b)[i] += r * x[i]
	}
}

// Decay old contributions in A and b (soft forgetting)
func applyDecay(arm *LinUCBArmState) {
	if decayRate <= 0 {
		return
	}
	decay := 1.0 - decayRate

	for i := range linUCBFeatureDim {
		for j := range linUCBFeatureDim {
			arm.A[i][j] *= decay
		}
		arm.B[i] *= decay
	}

	if arm.Count > 0 {
		arm.Count = int(float64(arm.Count) * decay)
	}
}

// Invert 4x4 matrix using Gauss-Jordan.
func invert4x4(A [linUCBFeatureDim][linUCBFeatureDim]float64) ([linUCBFeatureDim][linUCBFeatureDim]float64, error) {
	var aug [linUCBFeatureDim][2 * linUCBFeatureDim]float64

	// Build augmented [A | I]
	for i := range linUCBFeatureDim {
		for j := range linUCBFeatureDim {
			aug[i][j] = A[i][j]
		}
		aug[i][linUCBFeatureDim+i] = 1.0
	}

	// Gauss–Jordan elimination
	for col := range linUCBFeatureDim {
		pivot := aug[col][col]
		if math.Abs(pivot) < 1e-9 {
			return [linUCBFeatureDim][linUCBFeatureDim]float64{}, fmt.Errorf("matrix is singular")
		}

		// Normalize pivot row
		for j := range 2 * linUCBFeatureDim {
			aug[col][j] /= pivot
		}

		// Eliminate other rows
		for i := range linUCBFeatureDim {
			if i == col {
				continue
			}
			factor := aug[i][col]
			for j := range 2 * linUCBFeatureDim {
				aug[i][j] -= factor * aug[col][j]
			}
		}
	}

	// Extract inverse
	var inv [linUCBFeatureDim][linUCBFeatureDim]float64
	for i := range linUCBFeatureDim {
		for j := range linUCBFeatureDim {
			inv[i][j] = aug[i][linUCBFeatureDim+j]
		}
	}
	return inv, nil
}
