package config

import (
	"github.com/caarlos0/env/v11"
	"log"
)

type Config struct {
	AppEnv        string `env:"APP_ENV,notEmpty"`
	APIAddr       string `env:"API_ADDR,notEmpty"`
	SchedAddr     string `env:"SCHED_ADDR,notEmpty"`
	PostgresDSN   string `env:"POSTGRES_DSN,notEmpty"`
	RedisAddr     string `env:"REDIS_ADDR,notEmpty"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	JWTSigningKey string `env:"JWT_SIGNING_KEY,notEmpty"`
	DefaultVT     int    `env:"DEFAULT_VISIBILITY_TIMEOUT_SEC" envDefault:"60"`
}

func Load() Config {
	var c Config
	if err := env.Parse(&c); err != nil {
		log.Fatal(err)
	}
	return c
}
