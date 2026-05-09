package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server ServerConfig `env-prefix:"SERVER_"`
	Log    LogConfig    `env-prefix:"LOG_"`
}

type ServerConfig struct {
	Host string `env:"HOST" env-default:"0.0.0.0" env-description:"Server host"`
	Port int    `env:"PORT" env-default:"8080" env-description:"Server port"`
}

type LogConfig struct {
	Level  string `env:"LEVEL" env-default:"info" env-description:"Log level"`
	Format string `env:"FORMAT" env-default:"text" env-description:"Log format"`
}

func Load() (*Config, error) {
	var cfg Config

	err := cleanenv.ReadConfig(".env", &cfg)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return nil, fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	return &cfg, nil
}
