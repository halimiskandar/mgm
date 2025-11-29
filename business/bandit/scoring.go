package bandit

import (
	"math"
	"math/rand"
)

// ucbScore = theta·x + alpha * sqrt(x^T A^-1 x)
func ucbScore(theta, x [linUCBFeatureDim]float64, AInv [linUCBFeatureDim][linUCBFeatureDim]float64, alpha float64) float64 {
	mean := dot(theta, x)
	tmp := matVecMul(AInv, x)
	uncertainty := math.Sqrt(dot(x, tmp))
	return mean + alpha*uncertainty
}

// thompsonScore: diagonal Gaussian sampling of theta
func thompsonScore(theta, x [linUCBFeatureDim]float64, AInv [linUCBFeatureDim][linUCBFeatureDim]float64) float64 {
	var thetaSample [linUCBFeatureDim]float64
	for i := 0; i < linUCBFeatureDim; i++ {
		varVar := AInv[i][i]
		if varVar < 0 {
			varVar = 0
		}
		std := math.Sqrt(varVar)
		thetaSample[i] = theta[i] + rand.NormFloat64()*std
	}
	return dot(thetaSample, x)
}
