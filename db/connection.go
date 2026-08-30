package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config holds database configuration
type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	Database     string
	SSLMode      string
	MaxOpenConns int           // Maximum number of open connections (default: 25)
	MaxIdleConns int           // Maximum number of idle connections (default: 25)
	MaxLifetime  time.Duration // Maximum connection lifetime (default: 5 minutes)
	MaxIdleTime  time.Duration // Maximum connection idle time (default: 5 minutes)
}

// Connect establishes a database connection with the given configuration
func Connect(config Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		config.SSLMode,
	)

	// Apply sensible defaults if not configured
	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 25
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 25
	}
	if config.MaxLifetime == 0 {
		config.MaxLifetime = 5 * time.Minute
	}
	if config.MaxIdleTime == 0 {
		config.MaxIdleTime = 5 * time.Minute
	}

	// Validate configuration
	if config.MaxIdleConns > config.MaxOpenConns {
		return nil, fmt.Errorf("MaxIdleConns (%d) cannot exceed MaxOpenConns (%d)",
			config.MaxIdleConns, config.MaxOpenConns)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.MaxLifetime)
	db.SetConnMaxIdleTime(config.MaxIdleTime)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// HealthCheck performs a basic health check on the database
func HealthCheck(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}
