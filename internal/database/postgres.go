package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
	ConnectionMaxIdleTime time.Duration
}

func DefaultConfig() Config {
	return Config{
		MaxOpenConnections:    25,
		MaxIdleConnections:    10,
		ConnectionMaxLifetime: 30 * time.Minute,
		ConnectionMaxIdleTime: 5 * time.Minute,
	}
}

func (c Config) Validate() error {
	if c.MaxOpenConnections <= 0 {
		return errors.New("max open connections must be greater than zero")
	}

	if c.MaxIdleConnections < 0 {
		return errors.New("max idle connections cannot be negative")
	}

	if c.MaxIdleConnections > c.MaxOpenConnections {
		return errors.New("max idle connections cannot exceed max open connections")
	}

	if c.ConnectionMaxLifetime <= 0 {
		return errors.New("connection max lifetime must be greater than zero")
	}

	if c.ConnectionMaxIdleTime <= 0 {
		return errors.New("connection max idle time must be greater than zero")
	}

	return nil
}

func Open(ctx context.Context, connectionString string, cfg Config) (*sql.DB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database config: %w", err)
	}

	if connectionString == "" {
		return nil, errors.New("database connection string cannot be empty")
	}

	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.MaxIdleConnections)
	db.SetConnMaxLifetime(cfg.ConnectionMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnectionMaxIdleTime)

	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}

		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}
