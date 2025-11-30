package bandit

import (
	"fmt"
	"math"
)

const decayRate = 0.001 // soft forgetting

// y = A * x
func matVecMul(A [linUCBFeatureDim][linUCBFeatureDim]float64, x [linUCBFeatureDim]float64) [linUCBFeatureDim]float64 {
	var y [linUCBFeatureDim]float64
	for i := 0; i < linUCBFeatureDim; i++ {
		sum := 0.0
		for j := 0; j < linUCBFeatureDim; j++ {
			sum += A[i][j] * x[j]
		}
		y[i] = sum
	}
	return y
}

func dot(a, b [linUCBFeatureDim]float64) float64 {
	sum := 0.0
	for i := 0; i < linUCBFeatureDim; i++ {
		sum += a[i] * b[i]
	}
	return sum
}

// A := A + x x^T
func addOuter(A *[linUCBFeatureDim][linUCBFeatureDim]float64, x [linUCBFeatureDim]float64) {
	for i := 0; i < linUCBFeatureDim; i++ {
		for j := 0; j < linUCBFeatureDim; j++ {
			(*A)[i][j] += x[i] * x[j]
		}
	}
}

// b := b + r x
func addScaled(b *[linUCBFeatureDim]float64, x [linUCBFeatureDim]float64, r float64) {
	for i := 0; i < linUCBFeatureDim; i++ {
		(*b)[i] += r * x[i]
	}
}

// Decay old contributions in A and b (soft forgetting)
func applyDecay(arm *LinUCBArmState) {
	if decayRate <= 0 {
		return
	}
	decay := 1.0 - decayRate

	for i := 0; i < linUCBFeatureDim; i++ {
		for j := 0; j < linUCBFeatureDim; j++ {
			arm.A[i][j] *= decay
		}
		arm.B[i] *= decay
	}

	if arm.Count > 0 {
		arm.Count = int(float64(arm.Count) * decay)
	}
}

// "invert4x4" now inverts a linUCBFeatureDim x linUCBFeatureDim matrix using Gauss–Jordan.

func invert4x4(A [linUCBFeatureDim][linUCBFeatureDim]float64) ([linUCBFeatureDim][linUCBFeatureDim]float64, error) {
	var aug [linUCBFeatureDim][2 * linUCBFeatureDim]float64

	// Build augmented [A | I]
	for i := 0; i < linUCBFeatureDim; i++ {
		for j := 0; j < linUCBFeatureDim; j++ {
			aug[i][j] = A[i][j]
		}
		aug[i][linUCBFeatureDim+i] = 1.0
	}

	// Gauss–Jordan elimination
	for col := 0; col < linUCBFeatureDim; col++ {
		pivot := aug[col][col]
		if math.Abs(pivot) < 1e-9 {
			return [linUCBFeatureDim][linUCBFeatureDim]float64{}, fmt.Errorf("matrix is singular")
		}

		// Normalize pivot row
		for j := 0; j < 2*linUCBFeatureDim; j++ {
			aug[col][j] /= pivot
		}

		// Eliminate other rows
		for i := 0; i < linUCBFeatureDim; i++ {
			if i == col {
				continue
			}
			factor := aug[i][col]
			for j := 0; j < 2*linUCBFeatureDim; j++ {
				aug[i][j] -= factor * aug[col][j]
			}
		}
	}

	// Extract inverse
	var inv [linUCBFeatureDim][linUCBFeatureDim]float64
	for i := 0; i < linUCBFeatureDim; i++ {
		for j := 0; j < linUCBFeatureDim; j++ {
			inv[i][j] = aug[i][linUCBFeatureDim+j]
		}
	}
	return inv, nil
}
