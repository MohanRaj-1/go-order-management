package config

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	keys := []string{
		"HTTP_ADDR",
		"HTTP_READ_HEADER_TIMEOUT",
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"HTTP_SHUTDOWN_TIMEOUT",
		"LOG_LEVEL",
	}

	// Save the original values and remove the variables for this test.
	for _, key := range keys {
		value, existed := os.LookupEnv(key)

		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("failed to unset %s: %v", key, err)
		}

		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(key, value)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("expected address :8080, got %q", cfg.HTTPAddr)
	}
	if cfg.HTTPReadHeaderTimeout != 5*time.Second {
		t.Errorf("expected read header timeout 5s, got %s", cfg.HTTPReadHeaderTimeout)
	}
	if cfg.HTTPReadTimeout != 15*time.Second {
		t.Errorf("expected read timeout 15s, got %s", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != 30*time.Second {
		t.Errorf("expected write timeout 30s, got %s", cfg.HTTPWriteTimeout)
	}
	if cfg.HTTPIdleTimeout != 60*time.Second {
		t.Errorf("expected idle timeout 60s, got %s", cfg.HTTPIdleTimeout)
	}
	if cfg.HTTPShutdownTimeout != 10*time.Second {
		t.Errorf("expected shutdown timeout 10s, got %s", cfg.HTTPShutdownTimeout)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("expected log level INFO, got %s", cfg.LogLevel)
	}
}

func TestLoadCustomReadTimeout(t *testing.T) {
	t.Setenv("HTTP_READ_TIMEOUT", "20s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.HTTPReadTimeout != 20*time.Second {
		t.Errorf("expected read timeout 20s, got %s", cfg.HTTPReadTimeout)
	}
}

func TestLoadInvalidReadTimeout(t *testing.T) {
	t.Setenv("HTTP_READ_TIMEOUT", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error for invalid HTTP_READ_TIMEOUT")
	}
}

func TestValidateRejectsEmptyAddress(t *testing.T) {
	cfg := Config{
		HTTPAddr:              "   ",
		HTTPReadHeaderTimeout: 5 * time.Second,
		HTTPReadTimeout:       15 * time.Second,
		HTTPWriteTimeout:      30 * time.Second,
		HTTPIdleTimeout:       60 * time.Second,
		HTTPShutdownTimeout:   10 * time.Second,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected an error for an empty HTTP address")
	}
}

func TestValidateRejectsZeroReadTimeout(t *testing.T) {
	cfg := Config{
		HTTPAddr:              ":8080",
		HTTPReadHeaderTimeout: 5 * time.Second,
		HTTPReadTimeout:       0,
		HTTPWriteTimeout:      30 * time.Second,
		HTTPIdleTimeout:       60 * time.Second,
		HTTPShutdownTimeout:   10 * time.Second,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected an error for a zero HTTP read timeout")
	}
}

func TestLoadRejectsZeroReadTimeout(t *testing.T) {
	t.Setenv("HTTP_READ_TIMEOUT", "0s")

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load() to reject a zero HTTP read timeout")
	}
}

func TestLoadLogLevels(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  slog.Level
	}{
		{"debug", "DEBUG", slog.LevelDebug},
		{"info", "INFO", slog.LevelInfo},
		{"warn", "WARN", slog.LevelWarn},
		{"error", "ERROR", slog.LevelError},
		{"lowercase debug", "debug", slog.LevelDebug},
		{"trimmed debug", " Debug ", slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tt.value)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.LogLevel != tt.want {
				t.Errorf("expected log level %s, got %s", tt.want, cfg.LogLevel)
			}
		})
	}
}

func TestLoadRejectsUnsupportedLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "TRACE")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for unsupported log level")
	}
}
