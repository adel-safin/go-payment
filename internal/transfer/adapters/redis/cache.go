package rediscache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(addr, password string) *Cache {
	return &Cache{
		rdb: redis.NewClient(&redis.Options{Addr: addr, Password: password}),
		ttl: 24 * time.Hour,
	}
}

func (c *Cache) Client() *redis.Client { return c.rdb }

func (c *Cache) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := c.rdb.Get(ctx, redisKey(key)).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (c *Cache) Set(ctx context.Context, key, transferID string) error {
	return c.rdb.Set(ctx, redisKey(key), transferID, c.ttl).Err()
}

func redisKey(key string) string {
	return fmt.Sprintf("idempotency:transfer:%s", key)
}
