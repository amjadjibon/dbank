package conf

import (
	env "github.com/caarlos0/env/v11"
)

type Config struct {
	LogLevel string `env:"LOG_LEVEL" envDefault:"debug"`
}

func NewConfig() *Config {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		panic(err)
	}

	return &cfg
}
