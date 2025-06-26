package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	Ctx context.Context = context.Background()

	RedisClient RedisClientInterface
)

type RedisClientInterface interface {
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Decr(ctx context.Context, key string) *redis.IntCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
}

func InitRedis(addr string) error {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	if _, err := client.Ping(Ctx).Result(); err != nil {
		return err
	}
	RedisClient = client
	return nil
}

func Get(key string) (string, error) {
	return RedisClient.Get(Ctx, key).Result()
}

func Set(key string, value string, ttl time.Duration) error {
	return RedisClient.Set(Ctx, key, value, ttl).Err()
}

func Delete(keys ...string) error {
	if RedisClient == nil {
		return nil
	}
	return RedisClient.Del(Ctx, keys...).Err()
}
