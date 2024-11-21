package config

import "os"

type httpConfig struct {
	Host          string
	Port          string
	ExposeAddress string
}

func LoadHTTPConfig() httpConfig {
	return httpConfig{
		Host:          os.Getenv("HOST"),
		Port:          os.Getenv("PORT"),
		ExposeAddress: os.Getenv("EXPOSE_ADDRESS"),
	}
}
