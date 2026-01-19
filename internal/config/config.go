package config

import (
	"awesomeProject/pkg/logger"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Tick int `env:"TICK" env-default:"1"`
}

func NewConfig() *Config {
	var cfg Config
	err := cleanenv.ReadConfig(".env", &cfg)
	if err != nil {
		logger.Logger.Println("Error reading config:", err)
	}
	return &cfg
}
