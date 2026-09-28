package usecase

import "context"

// IdempotencyStore prevents duplicate side effects under at-least-once redelivery.
type IdempotencyStore interface {
	// Claim reserves key for processing. claimed=false means already processed/in-flight.
	Claim(ctx context.Context, key string) (claimed bool, err error)
	// Release allows a later retry after a failed attempt.
	Release(ctx context.Context, key string) error
}
