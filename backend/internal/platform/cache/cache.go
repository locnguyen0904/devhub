// Package cache owns the Redis client.
package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client so callers depend on this package instead of
// go-redis directly.
type Client struct {
	rdb *redis.Client
}

// New builds a Redis client from a connection URL. Like database.New it does
// not connect eagerly, keeping startup usable without a live Redis.
func New(url string) (*Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &Client{rdb: redis.NewClient(opt)}, nil
}

// Ping reports whether Redis answers.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

// Close shuts the client down.
func (c *Client) Close() error {
	if err := c.rdb.Close(); err != nil {
		return fmt.Errorf("close redis client: %w", err)
	}
	return nil
}
