package config

import (
	d "awesomeProject/internal/daemon/config"
	"awesomeProject/pkg/logger"
	"os"

	"gopkg.in/yaml.v3"
)

const FileName = "config.yaml"

type Config struct {
	Tick         int           `yaml:"tick"`
	LoggerConfig logger.Config `yaml:"logger"`
	Daemon       d.Config      `yaml:"daemon"`
	UI           UIConfig      `yaml:"ui"`
}

func DefaultConfig() Config {
	return Config{
		Tick:         1,
		LoggerConfig: logger.DefaultConfig(),
		Daemon:       d.DefaultConfig(),
		UI:           DefaultUIConfig(),
	}
}

func ReadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	if len(data) == 0 {
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func NewConfig() *Config {
	cfg, err := ReadConfig(FileName)
	if err != nil {
		return &Config{}
	}
	return &cfg
}
