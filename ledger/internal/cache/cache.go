package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/97Dmitry/go-hw/ledger/internal/db"
	"github.com/redis/go-redis/v9"
)

func Open(ctx context.Context) (*redis.Client, error) {
	n, err := strconv.Atoi(db.Env("REDIS_DB", "0"))
	if err != nil || n < 0 {
		return nil, fmt.Errorf("REDIS_DB must be a non-negative integer")
	}
	client := redis.NewClient(&redis.Options{Addr: db.Env("REDIS_ADDR", "localhost:6379"), DB: n, Password: os.Getenv("REDIS_PASSWORD"), DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second, MaxRetries: -1, ContextTimeoutEnabled: true})
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("connect Redis (check REDIS_*): %w", err)
	}
	return client, nil
}
