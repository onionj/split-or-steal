package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func IsLimited(rdb *redis.Client, key string, Limit int, exp time.Duration) (bool, error) {
	_, err := rdb.SetNX(context.Background(), key, Limit, exp).Result()
	if err != nil {
		logrus.Error("IsLimited error", err)
		return false, err
	}

	tokens, err := rdb.Get(context.Background(), key).Int()
	if err != nil {
		logrus.Error("IsLimited error", err)
		return false, err
	}
	if tokens <= 0 {
		return true, nil
	}
	return false, nil
}

func BurnToken(rdb *redis.Client, key string) {
	_, err := rdb.Decr(context.Background(), key).Result()
	if err != nil {
		logrus.Error("BurnToken error", err)
	}
}
