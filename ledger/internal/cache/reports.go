package cache

import (
	"context"
	"github.com/97Dmitry/go-hw/ledger/internal/domain"
	"github.com/redis/go-redis/v9"
	"time"
)

// Reports adapts Redis to the service's cache contract.
type Reports struct{ Client *redis.Client }

func (r Reports) Get(ctx context.Context, key string) ([]byte, error) {
	return r.Client.Get(ctx, key).Bytes()
}
func (r Reports) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return r.Client.Set(ctx, key, data, ttl).Err()
}

var _ domain.ReportCache = Reports{}
