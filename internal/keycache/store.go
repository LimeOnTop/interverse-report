package keycache

import (
	"context"
	"fmt"
	"time"

	"github.com/LimeOnTop/interverse-report/internal/usecase"
	"github.com/redis/go-redis/v9"
)

// Store caches idempotency keys in Redis (SETNX / DEL).
type Store struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

func NewStore(client *redis.Client, ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 72 * time.Hour
	}
	return &Store{
		client: client,
		ttl:    ttl,
		prefix: "report:kafka:idempotency:",
	}
}

var _ usecase.IdempotencyStore = (*Store)(nil)

func (s *Store) Claim(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("idempotency key is empty")
	}
	ok, err := s.client.SetNX(ctx, s.prefix+key, "1", s.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("idempotency claim: %w", err)
	}
	return ok, nil
}

func (s *Store) Release(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	if err := s.client.Del(ctx, s.prefix+key).Err(); err != nil {
		return fmt.Errorf("idempotency release: %w", err)
	}
	return nil
}
