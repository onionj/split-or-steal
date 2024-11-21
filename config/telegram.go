package config

import "os"

type telegramConfig struct {
	Token string
}

func LoadTelegramConfig() telegramConfig {
	return telegramConfig{
		Token: os.Getenv("TOKEN"),
	}
}
