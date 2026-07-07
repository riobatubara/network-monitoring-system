package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

// NewClient Redis cache cluster.
func NewClient(addr string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // Default to no password
		DB:       0,  // Default database
	})

	// Verify connection is active
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

// Close safely shuts down the Redis connection.
func (c *Client) Close() error {
	if c.rdb != nil {
		return c.rdb.Close()
	}
	return nil
}

// TakeToken implements a distributed token bucket mechanism for rate limiting device queries.
// It returns true if the check is allowed, or false if the rate limit is exceeded.
func (c *Client) TakeToken(ctx context.Context, target string, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("ratelimit:%s", target)

	// Increment the counter for this target device
	count, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis rate limiter error: %w", err)
	}

	// Set expiration on the first hit to establish the time window
	if count == 1 {
		if err := c.rdb.Expire(ctx, key, window).Err(); err != nil {
			return false, fmt.Errorf("redis expire tracker error: %w", err)
		}
	}

	// Deny action if target device checks exceed the maximum safety threshold
	if count > int64(limit) {
		return false, nil
	}

	return true, nil
}
