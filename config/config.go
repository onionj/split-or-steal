package config

import (
	"log"

	"github.com/joho/godotenv"
)

type ConfigT struct {
	HTTP     httpConfig
	Redis    redisConfig
	Telegram telegramConfig
}

var GlobalConfig ConfigT

func NewConfig(envFile string) ConfigT {
	if envFile == "" {
		envFile = ".env"
	}
	err := godotenv.Load(envFile)
	if err != nil {
		log.Println("Error loading .env file", err)
	}
	GlobalConfig = ConfigT{
		HTTP:     LoadHTTPConfig(),
		Redis:    LoadRedisConfig(),
		Telegram: LoadTelegramConfig(),
	}

	return GlobalConfig
}
