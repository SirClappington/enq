package config

import (
	"log"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppEnv                 string `env:"APP_ENV" envDefault:"local"`
	APIAddr                string `env:"API_ADDR" envDefault:":8080"`
	SchedAddr              string `env:"SCHED_ADDR" envDefault:":8081"`
	PostgresDSN            string `env:"POSTGRES_DSN,notEmpty"`
	RedisAddr              string `env:"REDIS_ADDR,notEmpty"`
	RedisPassword          string `env:"REDIS_PASSWORD"`
	JWTSigningKey          string `env:"JWT_SIGNING_KEY" envDefault:"dev-signing-key"`
	DefaultVisibilityTOSec int    `env:"DEFAULT_VISIBILITY_TIMEOUT_SEC" envDefault:"60"`
}

func Load() Config {
	var c Config
	if err := env.Parse(&c); err != nil {
		log.Fatal(err)
	}
	return c
}
