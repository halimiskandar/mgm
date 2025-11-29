package bandit

import "context"

// EligibilityChecker decides if a given product is allowed to be recommended
// for a user in a given slot (stock, visibility, location)
type EligibilityChecker interface {
	IsEligible(ctx context.Context, userID uint, productID uint64, slot string) (bool, error)
}

// NoopEligibilityChecker is the default implementation that allows everything.
type NoopEligibilityChecker struct{}

func (NoopEligibilityChecker) IsEligible(ctx context.Context, userID uint, productID uint64, slot string) (bool, error) {
	return true, nil
}
