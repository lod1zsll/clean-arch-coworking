package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppPort  string `env:"APP_PORT"  envDefault:"8080"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	TopicIn  string `env:"TOPIC_IN" envDefault:"events"`
	TopicOut string `env:"TOPIC_OUT" envDefault:"events"`

	PostgresHost     string `env:"POSTGRES_HOST"     envDefault:"localhost"`
	PostgresPort     string `env:"POSTGRES_PORT"     envDefault:"5432"`
	PostgresDB       string `env:"POSTGRES_DB"       envDefault:"coworking"`
	PostgresUser     string `env:"POSTGRES_USER"     envDefault:"coworking"`
	PostgresPassword string `env:"POSTGRES_PASSWORD" envDefault:"secret"`
}

// PostgresDSN builds a connection string from individual Postgres fields.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.PostgresUser, c.PostgresPassword,
		c.PostgresHost, c.PostgresPort, c.PostgresDB,
	)
}

// Load parses environment variables into Config using struct tags.
func Load() (*Config, error) {
	_ = godotenv.Load() // for local testing

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
