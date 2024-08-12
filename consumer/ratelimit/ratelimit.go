package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	ShouldWait(ctx context.Context, key string) (time.Duration, error)
}

type noRateLimiter struct{}

func (r *noRateLimiter) ShouldWait(_ context.Context, _ string) (time.Duration, error) {
	return 0, nil
}

type rateLimiter struct {
	redisClient *redis.Client
	duration    time.Duration
	limit       int
}

func NewRateLimiter(redisClient *redis.Client, duration time.Duration, limit int) RateLimiter {
	if duration <= 0 || limit <= 0 {
		return &noRateLimiter{}
	}
	return &rateLimiter{
		redisClient: redisClient,
		duration:    duration,
		limit:       limit,
	}
}

func (r *rateLimiter) ShouldWait(ctx context.Context, key string) (time.Duration, error) {
	now := time.Now()
	start := now.Add(-r.duration)
	nowStr, startStr := fmt.Sprint(now.UnixMicro()), fmt.Sprint(start.UnixMicro())

	pipeline := r.redisClient.TxPipeline()
	pipeline.ZRemRangeByScore(ctx, key, "0", startStr)
	pipeline.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixMicro()), Member: nowStr})
	pipeline.Expire(ctx, key, r.duration)
	scores := pipeline.ZRangeWithScores(ctx, key, 0, -1)

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return 0, err
	}

	if len(scores.Val()) > r.limit {
		z := scores.Val()[0]
		last := time.UnixMicro(int64(z.Score))
		expiration := last.Add(r.duration)
		timeToWait := expiration.Sub(now)
		return timeToWait, err
	}
	return 0, nil
}
