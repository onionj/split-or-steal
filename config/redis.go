package config

import (
	"os"
)

type redisConfig struct {
	DB       int
	Password string
	Host     string
	Port     string
}

func LoadRedisConfig() redisConfig {

	return redisConfig{
		DB:       0,
		Password: os.Getenv("REDIS_PASSWORD"),
		Host:     os.Getenv("REDIS_HOST"),
		Port:     os.Getenv("REDIS_PORT"),
	}
}
