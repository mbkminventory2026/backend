package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

const (
	envFileName            = ".env"
	defaultServerPort      = "8080"
	defaultDBPort          = "5432"
	defaultDBSSLMode       = "disable"
	defaultCORSAllowOrigin = "*"
	defaultDBMaxConns      = int32(20)
	defaultDBMinConns      = int32(2)
	defaultDBConnLifetime  = 30
	defaultDBConnIdleTime  = 10
	defaultDBHealthPeriod  = 30
	defaultDBConnectTO     = 5
)

// Config stores all application settings loaded from .env or system environment.
type Config struct {
	ServerPort      string `mapstructure:"SERVER_PORT"`
	CORSAllowOrigin string `mapstructure:"CORS_ALLOW_ORIGIN"`

	DBURL      string `mapstructure:"DB_URL"`
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBSSLMode  string `mapstructure:"DB_SSLMODE"`
	DBMaxConns int32  `mapstructure:"DB_MAX_CONNS"`
	DBMinConns int32  `mapstructure:"DB_MIN_CONNS"`

	DBMaxConnLifetimeMinutes int `mapstructure:"DB_MAX_CONN_LIFETIME_MINUTES"`
	DBMaxConnIdleTimeMinutes int `mapstructure:"DB_MAX_CONN_IDLE_TIME_MINUTES"`
	DBHealthCheckPeriodSec   int `mapstructure:"DB_HEALTH_CHECK_PERIOD_SECONDS"`
	DBConnectTimeoutSec      int `mapstructure:"DB_CONNECT_TIMEOUT_SECONDS"`

	JWTSecret       string `mapstructure:"JWT_SECRET"`
	TurnstileSecret string `mapstructure:"TURNSTILE_SECRET"`
}

// Load reads configuration from .env (if present) and environment variables.
func Load() (*Config, error) {
	viper.SetConfigFile(envFileName)
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	viper.SetDefault("SERVER_PORT", defaultServerPort)
	viper.SetDefault("DB_PORT", defaultDBPort)
	viper.SetDefault("DB_SSLMODE", defaultDBSSLMode)
	viper.SetDefault("CORS_ALLOW_ORIGIN", defaultCORSAllowOrigin)
	viper.SetDefault("DB_MAX_CONNS", defaultDBMaxConns)
	viper.SetDefault("DB_MIN_CONNS", defaultDBMinConns)
	viper.SetDefault("DB_MAX_CONN_LIFETIME_MINUTES", defaultDBConnLifetime)
	viper.SetDefault("DB_MAX_CONN_IDLE_TIME_MINUTES", defaultDBConnIdleTime)
	viper.SetDefault("DB_HEALTH_CHECK_PERIOD_SECONDS", defaultDBHealthPeriod)
	viper.SetDefault("DB_CONNECT_TIMEOUT_SECONDS", defaultDBConnectTO)

	if err := viper.ReadInConfig(); err != nil {
		var configNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configNotFound) {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

// DatabaseURL returns DB_URL when provided, otherwise builds a PostgreSQL DSN from DB_* values.
func (c *Config) DatabaseURL() string {
	if strings.TrimSpace(c.DBURL) != "" {
		return c.DBURL
	}

	user := url.QueryEscape(c.DBUser)
	password := url.QueryEscape(c.DBPassword)

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user,
		password,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}
