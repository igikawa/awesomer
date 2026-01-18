package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	Tick int `env:"TICK" envDefault:"5"`
}

func NewConfig() *Config {
	var cfg Config
	err := cleanenv.ReadConfig(".env", &cfg)
	if err != nil {
		panic(err)
	}
	return &cfg
}
