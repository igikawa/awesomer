package config

import (
	d "awesomeProject/internal/daemon/config"
	"awesomeProject/pkg/logger"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Tick         int `env:"TICK" env-default:"1"`
	LoggerConfig logger.Config
	Daemon       d.Config
}

func NewConfig() *Config {
	var cfg Config
	cleanenv.ReadConfig(".env", &cfg)
	return &cfg
}
