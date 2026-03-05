package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
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
// When .env is missing (e.g. inside Docker), it falls back to OS environment variables only.
func Load() (*Config, error) {
	viper.SetConfigFile(envFileName)
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// Explicitly bind all known keys so that viper.Unmarshal picks up
	// OS environment variables even when no .env file is present.
	// (AutomaticEnv only works with viper.Get, not with Unmarshal.)
	for _, key := range []string{
		"SERVER_PORT", "CORS_ALLOW_ORIGIN", "GIN_MODE",
		"DB_URL", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD",
		"DB_NAME", "DB_SSLMODE", "DB_MAX_CONNS", "DB_MIN_CONNS",
		"DB_MAX_CONN_LIFETIME_MINUTES", "DB_MAX_CONN_IDLE_TIME_MINUTES",
		"DB_HEALTH_CHECK_PERIOD_SECONDS", "DB_CONNECT_TIMEOUT_SECONDS",
		"JWT_SECRET", "TURNSTILE_SECRET",
	} {
		_ = viper.BindEnv(key)
	}

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
		// When using SetConfigFile with an explicit path, viper returns
		// *os.PathError (not ConfigFileNotFoundError) if the file is missing.
		// Both cases are safe to ignore — the app will use OS env vars instead.
		var configNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configNotFound) && !errors.Is(err, os.ErrNotExist) {
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
