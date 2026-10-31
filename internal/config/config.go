package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr              string
	HTTPReadHeaderTimeout time.Duration
	HTTPReadTimeout       time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration
	HTTPShutdownTimeout   time.Duration
	LogLevel              slog.Level
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr: ":8080",
	}

	var err error

	if cfg.HTTPReadHeaderTimeout, err = durationFromEnv(
		"HTTP_READ_HEADER_TIMEOUT", 5*time.Second,
	); err != nil {
		return Config{}, err
	}

	if cfg.HTTPReadTimeout, err = durationFromEnv(
		"HTTP_READ_TIMEOUT", 15*time.Second,
	); err != nil {
		return Config{}, err
	}

	if cfg.HTTPWriteTimeout, err = durationFromEnv(
		"HTTP_WRITE_TIMEOUT", 30*time.Second,
	); err != nil {
		return Config{}, err
	}

	if cfg.HTTPIdleTimeout, err = durationFromEnv(
		"HTTP_IDLE_TIMEOUT", 60*time.Second,
	); err != nil {
		return Config{}, err
	}

	if cfg.HTTPShutdownTimeout, err = durationFromEnv(
		"HTTP_SHUTDOWN_TIMEOUT", 10*time.Second,
	); err != nil {
		return Config{}, err
	}

	if addr, ok := os.LookupEnv("HTTP_ADDR"); ok {
		cfg.HTTPAddr = addr
	}

	if cfg.LogLevel, err = logLevelFromEnv(); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate configuration: %w", err)
	}

	return cfg, nil
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q: %w", key, value, err)
	}

	return duration, nil
}

func (cfg Config) Validate() error {
	if strings.TrimSpace(cfg.HTTPAddr) == "" {
		return fmt.Errorf("HTTP_ADDR must not be empty")
	}

	timeouts := []struct {
		name  string
		value time.Duration
	}{
		{"HTTP_READ_HEADER_TIMEOUT", cfg.HTTPReadHeaderTimeout},
		{"HTTP_READ_TIMEOUT", cfg.HTTPReadTimeout},
		{"HTTP_WRITE_TIMEOUT", cfg.HTTPWriteTimeout},
		{"HTTP_IDLE_TIMEOUT", cfg.HTTPIdleTimeout},
		{"HTTP_SHUTDOWN_TIMEOUT", cfg.HTTPShutdownTimeout},
	}

	for _, timeout := range timeouts {
		if timeout.value <= 0 {
			return fmt.Errorf("%s must be greater than zero", timeout.name)
		}
	}

	return nil
}

func logLevelFromEnv() (slog.Level, error) {
	value, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		return slog.LevelInfo, nil
	}

	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid LOG_LEVEL value %q: supported values are DEBUG, INFO, WARN, ERROR", value)
	}
}
