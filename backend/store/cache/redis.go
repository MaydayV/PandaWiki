package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/chaitin/panda-wiki/config"
)

type Cache struct {
	*redis.Client
}

func NewCache(config *config.Config) (*Cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password,
	})
	// test connection
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &Cache{
		Client: rdb,
	}, nil
}

func (cache *Cache) GetOrSet(ctx context.Context, key string, value interface{}, expiration time.Duration) (interface{}, error) {
	// 先 Get 后 Set 非原子：多实例同时冷启动会各自写入不同值。
	// 用 SetNX 保证只有一个实例的值生效，其余实例读取已存在的值。
	set, err := cache.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		return nil, err
	}
	if set {
		return value, nil
	}
	return cache.Get(ctx, key).Result()
}

// DeleteKeysWithPrefix 删除所有指定前缀的 key
func (cache *Cache) DeleteKeysWithPrefix(ctx context.Context, prefix string) error {
	iter := cache.Scan(ctx, 0, prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		if err := cache.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return nil
}

func (cache *Cache) AcquireLock(ctx context.Context, key string) bool {
	result, err := cache.SetNX(ctx, key, true, 10*time.Second).Result()
	if err != nil {
		return false
	}
	return result
}

func (cache *Cache) ReleaseLock(ctx context.Context, key string) bool {
	_, err := cache.Del(ctx, key).Result()
	return err == nil
}

var releaseLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

// AcquireLockWithTTL 获取带随机持有者 token 与 TTL 的分布式锁。
func (cache *Cache) AcquireLockWithTTL(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	return cache.SetNX(ctx, key, token, ttl).Result()
}

// ReleaseLockWithToken 仅在锁持有者 token 匹配时释放分布式锁。
func (cache *Cache) ReleaseLockWithToken(ctx context.Context, key, token string) error {
	return releaseLockScript.Run(ctx, cache.Client, []string{key}, token).Err()
}

// IncrWindow 原子递增并按首次写入设定窗口过期，用于简单滑动窗口限频
func (cache *Cache) IncrWindow(ctx context.Context, key string, window time.Duration) (int64, error) {
	pipe := cache.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}
