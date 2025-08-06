package cache

import (
	"context"

	"github.com/ferdzo/ferurl/utils"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type Cache struct {
	client *redis.Client
}

func NewRedisClient(config utils.RedisConfig) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     utils.RedisUrl(),
		Password: config.Password,
		DB:       0,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Cache{client: client}, nil
}

func (c *Cache) Get(key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *Cache) Set(key string, value string) error {
	return c.client.Set(ctx, key, value, 0).Err()
}
