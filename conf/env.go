package conf

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	LogLevel string `env:"LOG_LEVEL" envDefault:"debug"`
	Host     string `env:"HOST"      envDefault:"0.0.0.0"`
	HTTPPort int    `env:"HTTP_PORT" envDefault:"8080"`
	GrpcPort int    `env:"GRPC_PORT" envDefault:"50051"`
}

func NewConfig() *Config {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		panic(err)
	}

	return &cfg
}
