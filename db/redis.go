package db

import (
	"fmt"

	"github.com/onionj/trust/config"
	"github.com/redis/go-redis/v9"
)

func Init(cfg config.ConfigT) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:       fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password:   cfg.Redis.Password,
		DB:         cfg.Redis.DB,
		ClientName: "trust",
	})
}
