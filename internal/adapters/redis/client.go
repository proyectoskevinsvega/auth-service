package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addrs       []string
	Password    string
	DB          int
	ClusterMode bool
}

func NewClient(cfg Config) (redis.UniversalClient, error) {
	if cfg.ClusterMode {
		client := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.Addrs,
			Password: cfg.Password,
		})

		// Test connection
		if err := client.Ping(context.Background()).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis cluster: %w", err)
		}

		return client, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addrs[0],
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}
