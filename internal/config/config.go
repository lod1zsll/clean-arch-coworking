package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppPort  string `env:"APP_PORT"  envDefault:"8080"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	ShutdownTimeoutSec int64 `env:"SHUTDOWN_TIMEOUT_SEC" envDefault:"8"`

	TopicIn  string `env:"TOPIC_IN" envDefault:"events"`
	TopicOut string `env:"TOPIC_OUT" envDefault:"events"`

	NatsConnectString string `env:"NATS_DSN" envDefault:"nats://127.0.0.1:4222"`
	PostgresHost      string `env:"POSTGRES_HOST"     envDefault:"localhost"`
	PostgresPort      string `env:"POSTGRES_PORT"     envDefault:"5432"`
	PostgresDB        string `env:"POSTGRES_DB"       envDefault:"coworking"`
	PostgresUser      string `env:"POSTGRES_USER"     envDefault:"coworking"`
	PostgresPassword  string `env:"POSTGRES_PASSWORD" envDefault:"secret"`

	RedisHost     string `env:"REDIS_HOST" envDefault:"localhost"`
	RedisPort     string `env:"REDIS_PORT" envDefault:"6379"`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0"`
	RedisUser     string `env:"REDIS_USER" envDefault:""`
	RedisPassword string `env:"REDIS_PASSWORD" envDefault:"secret"`

	DedupTTLSec int64 `env:"DEDUPLICATION_TTL" envDefault:"0"`
}

// PostgresDSN builds a connection string from individual Postgres fields.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.PostgresUser, c.PostgresPassword,
		c.PostgresHost, c.PostgresPort, c.PostgresDB,
	)
}

func (c *Config) NatsDSN() string {
	return c.NatsConnectString
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
