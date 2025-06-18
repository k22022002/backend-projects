package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	Ctx         = context.Background()
	RedisClient *redis.Client
)

func InitRedis(addr string) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr: addr,
	})
	_, err := RedisClient.Ping(Ctx).Result()
	return err
}

func Get(key string) (string, error) {
	return RedisClient.Get(Ctx, key).Result()
}

func Set(key string, value string, ttl time.Duration) error {
	return RedisClient.Set(Ctx, key, value, ttl).Err()
}

func Delete(keys ...string) error {
	return RedisClient.Del(Ctx, keys...).Err()
}
