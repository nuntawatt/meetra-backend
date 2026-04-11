package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	eventUC "github.com/nuntawatt/meetra-backend/internal/usecase/event"
	userUC "github.com/nuntawatt/meetra-backend/internal/usecase/user"
)

// cacheRepo is a Redis implementation that satisfies both
// user.CacheRepository and event.CacheRepository interfaces.
type cacheRepo struct {
	client *redis.Client
}

// NewCacheRepo constructs a cacheRepo.
func NewCacheRepo(client *redis.Client) *cacheRepo {
	return &cacheRepo{client: client}
}

// AsUserCache casts the repo to the user.CacheRepository interface.
func (r *cacheRepo) AsUserCache() userUC.CacheRepository { return r }

// AsEventCache casts the repo to the event.CacheRepository interface.
func (r *cacheRepo) AsEventCache() eventUC.CacheRepository { return r }

// ——— Core operations —————————————————————————————————————————————————————————

// Get retrieves a value by key. Returns "" and nil if the key does not exist.
func (r *cacheRepo) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // cache miss is not an error
	}
	if err != nil {
		return "", fmt.Errorf("cacheRepo.Get %s: %w", key, err)
	}
	return val, nil
}

// Set stores a value as JSON with an optional TTL (0 = no expiry).
func (r *cacheRepo) Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error {
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	default:
		var err error
		data, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("cacheRepo.Set marshal %s: %w", key, err)
		}
	}

	ttl := time.Duration(ttlSeconds) * time.Second
	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("cacheRepo.Set %s: %w", key, err)
	}
	return nil
}

// Delete removes a key from the cache. Ignores non-existent keys.
func (r *cacheRepo) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil && err != redis.Nil {
		return fmt.Errorf("cacheRepo.Delete %s: %w", key, err)
	}
	return nil
}
