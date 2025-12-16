package utils

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	ErrRateLimited = errors.New("rate limited")
	// 全局限流器实例
	globalRateLimiter *RateLimiter
)

// RateLimiter 分布式限流器
type RateLimiter struct {
	client    redis.UniversalClient
	keyPrefix string
}

// LimitConfig 限流配置
type LimitConfig struct {
	WindowSize  time.Duration // 时间窗口大小
	MaxRequests int           // 窗口内最大请求数
}

// NewRateLimiter 创建限流器
func NewRateLimiter(client redis.UniversalClient, keyPrefix string) *RateLimiter {
	if keyPrefix == "" {
		keyPrefix = "rate_limit:"
	}
	return &RateLimiter{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// InitGlobalRateLimiter 初始化全局限流器
func InitGlobalRateLimiter(client redis.UniversalClient, keyPrefix string) {
	globalRateLimiter = NewRateLimiter(client, keyPrefix)
}

// GetGlobalRateLimiter 获取全局限流器
func GetGlobalRateLimiter() *RateLimiter {
	return globalRateLimiter
}
func (rl *RateLimiter) SlidingWindowAllow(ctx context.Context, key string, config LimitConfig) (bool, error) {
	now := time.Now().UnixMilli()
	windowKey := rl.keyPrefix + "sliding:" + key

	// 使用ZSET存储请求时间戳
	member := fmt.Sprintf("%d:%d", now, rand.Int63())
	windowSize := config.WindowSize.Milliseconds()

	pipe := rl.client.TxPipeline()

	// 添加当前请求时间戳
	pipe.ZAdd(ctx, windowKey, &redis.Z{
		Score:  float64(now),
		Member: member,
	})

	// 移除窗口外的请求
	pipe.ZRemRangeByScore(ctx, windowKey, "0", fmt.Sprintf("%d", now-windowSize))

	// 获取窗口内请求数
	countCmd := pipe.ZCard(ctx, windowKey)

	// 设置过期时间
	pipe.Expire(ctx, windowKey, config.WindowSize*2)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("redis pipeline failed: %w", err)
	}

	count, err := countCmd.Result()
	if err != nil {
		return false, fmt.Errorf("get count failed: %w", err)
	}

	// 如果超过限制，移除当前请求
	if count > int64(config.MaxRequests) {
		rl.client.ZRem(ctx, windowKey, member)
		return false, nil
	}

	return true, nil
}

// Allow 通用限流接口
func (rl *RateLimiter) Allow(ctx context.Context, key string, config LimitConfig) error {
	allowed, err := rl.SlidingWindowAllow(ctx, key, config)
	if err != nil {
		return err
	}

	if !allowed {
		return ErrRateLimited
	}

	return nil
}

// AllowWithRetry 限流检查，失败时等待重试
func (rl *RateLimiter) AllowWithRetry(ctx context.Context, key string, config LimitConfig, maxRetries int, retryDelay time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		err := rl.Allow(ctx, key, config)
		if err == nil {
			return nil
		}

		if err != ErrRateLimited {
			return err
		}

		if i < maxRetries-1 {
			select {
			case <-time.After(retryDelay):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return ErrRateLimited
}
