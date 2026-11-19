package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/cache"
)

type RateLimiter struct {
	logger *log.Logger
	cache  *cache.Cache
	// scope 隔离不同端点的限速 key：匿名公开端点与管理员登录必须分桶，
	// 否则匿名者可故意锁定同 IP 的管理员登录（跨端点 DoS）
	scope string
}

func NewRateLimiter(logger *log.Logger, cache *cache.Cache) *RateLimiter {
	return &RateLimiter{
		logger: logger,
		cache:  cache,
		scope:  "login",
	}
}

// NewRateLimiterWithScope 创建带独立 key 前缀的限速器（不同端点互不影响）
func NewRateLimiterWithScope(logger *log.Logger, cache *cache.Cache, scope string) *RateLimiter {
	return &RateLimiter{
		logger: logger,
		cache:  cache,
		scope:  scope,
	}
}

const (
	LockThreshold1    = 5  // 第一次锁定阈值
	LockThreshold2    = 10 // 第二次锁定阈值
	LockThreshold3    = 15 // 第三次锁定阈值
	AttemptsKeyExpiry = 24 * time.Hour
)

// CheckIPLocked checks if the IP is currently locked
// Returns:
// - bool: whether the IP is locked
// - time.Duration: remaining lockout duration
func (r *RateLimiter) CheckIPLocked(ctx context.Context, ip string) (bool, time.Duration) {
	lockKey := fmt.Sprintf("%s_lock:%s", r.scope, ip)

	ttl, err := r.cache.TTL(ctx, lockKey).Result()
	if err != nil {
		r.logger.Error("failed to check lock status", "error", err, "ip", ip)
		return false, 0
	}

	if ttl > 0 {
		return true, ttl
	}

	return false, 0
}

func (r *RateLimiter) LockAttempt(ctx context.Context, ip string) {
	attemptsKey := fmt.Sprintf("%s_attempts:%s", r.scope, ip)
	lockKey := fmt.Sprintf("%s_lock:%s", r.scope, ip)

	attempts, err := r.cache.Incr(ctx, attemptsKey).Result()
	if err != nil {
		r.logger.Error("failed to increment attempts", "error", err, "ip", ip)
		return
	}

	if err := r.cache.Expire(ctx, attemptsKey, AttemptsKeyExpiry).Err(); err != nil {
		r.logger.Error("failed to set expiry on attempts key", "error", err, "ip", ip)
	}

	var lockDuration time.Duration

	if attempts%5 == 0 {
		switch {
		case attempts == LockThreshold1:
			lockDuration = time.Minute
		case attempts == LockThreshold2:
			lockDuration = 15 * time.Minute
		case attempts >= LockThreshold3:
			lockDuration = time.Hour
		}
		if lockDuration > 0 {
			if err := r.cache.Set(ctx, lockKey, 1, lockDuration).Err(); err != nil {
				r.logger.Error("failed to set lock key", "error", err, "ip", ip)
				return
			}
			r.logger.Info("IP has been locked", "ip", ip, "lockDuration", lockDuration)
		}
	}
}

// ResetLoginAttempts resets the login attempt counter and lock for an IP
func (r *RateLimiter) ResetLoginAttempts(ctx context.Context, ip string) error {
	attemptsKey := fmt.Sprintf("%s_attempts:%s", r.scope, ip)
	lockKey := fmt.Sprintf("%s_lock:%s", r.scope, ip)

	pipe := r.cache.Pipeline()
	pipe.Del(ctx, attemptsKey)
	pipe.Del(ctx, lockKey)
	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("failed to reset login attempts: %w", err)
	}
	return nil
}
