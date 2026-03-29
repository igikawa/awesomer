package config

import (
	d "awesomeProject/internal/daemon/config"
	"awesomeProject/pkg/logger"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	AppName          = "awesomer"
	FileName         = "config.yaml"
	SystemConfigPath = "/etc/awesomer/config.yaml"
)

var (
	userConfigDirFn = os.UserConfigDir
	currentEUIDFn   = os.Geteuid
)

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

func UserConfigPath() (string, error) {
	configDir, err := userConfigDirFn()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}

	return filepath.Join(configDir, AppName, FileName), nil
}

func SearchPaths() ([]string, error) {
	if currentEUIDFn() == 0 {
		return []string{SystemConfigPath}, nil
	}

	userPath, err := UserConfigPath()
	if err != nil {
		return nil, err
	}

	return []string{userPath}, nil
}

func ResolveConfigPath() (string, error) {
	paths, err := SearchPaths()
	if err != nil {
		if len(paths) == 0 {
			return "", err
		}
	}

	for _, path := range paths {
		if _, statErr := os.Stat(path); statErr == nil {
			return path, nil
		}
	}

	if len(paths) == 0 {
		return "", fmt.Errorf("no config search paths available")
	}

	return paths[0], err
}

func NewConfig() *Config {
	path, err := ResolveConfigPath()
	if err != nil {
		return &Config{}
	}

	cfg, err := ReadConfig(path)
	if err != nil {
		return &Config{}
	}
	return &cfg
}
