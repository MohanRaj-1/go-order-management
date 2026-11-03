package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid configuration",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "zero max open connections",
			config: Config{
				MaxOpenConnections:    0,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 30 * time.Minute,
				ConnectionMaxIdleTime: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "negative max idle connections",
			config: Config{
				MaxOpenConnections:    10,
				MaxIdleConnections:    -1,
				ConnectionMaxLifetime: 30 * time.Minute,
				ConnectionMaxIdleTime: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "max idle connections exceed max open connections",
			config: Config{
				MaxOpenConnections:    5,
				MaxIdleConnections:    10,
				ConnectionMaxLifetime: 30 * time.Minute,
				ConnectionMaxIdleTime: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "zero connection max lifetime",
			config: Config{
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 0,
				ConnectionMaxIdleTime: 5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "zero connection max idle time",
			config: Config{
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 30 * time.Minute,
				ConnectionMaxIdleTime: 0,
			},
			wantErr: true,
		},
		{
			name: "zero max idle connections is valid",
			config: Config{
				MaxOpenConnections:    10,
				MaxIdleConnections:    0,
				ConnectionMaxLifetime: 30 * time.Minute,
				ConnectionMaxIdleTime: 5 * time.Minute,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxOpenConnections != 25 {
		t.Errorf("MaxOpenConnections = %d, want 25", cfg.MaxOpenConnections)
	}

	if cfg.MaxIdleConnections != 10 {
		t.Errorf("MaxIdleConnections = %d, want 10", cfg.MaxIdleConnections)
	}

	if cfg.ConnectionMaxLifetime != 30*time.Minute {
		t.Errorf("ConnectionMaxLifetime = %v, want %v",
			cfg.ConnectionMaxLifetime, 30*time.Minute)
	}

	if cfg.ConnectionMaxIdleTime != 5*time.Minute {
		t.Errorf("ConnectionMaxIdleTime = %v, want %v",
			cfg.ConnectionMaxIdleTime, 5*time.Minute)
	}
}

func TestOpen(t *testing.T) {
	connectionString := os.Getenv("DATABASE_URL")
	if connectionString == "" {
		t.Skip("skipping integration test: DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, connectionString, DefaultConfig())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}
}
