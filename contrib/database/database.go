// Package database provides database utilities for the Quark framework.
// It wraps database/sql with helpful utilities for connection pooling,
// transaction management, and pagination.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// DB wraps sql.DB with additional utilities.
type DB struct {
	*sql.DB
	driver string
}

// Config holds database connection configuration.
type Config struct {
	Driver          string        `env:"DB_DRIVER" default:"postgres"`
	Host            string        `env:"DB_HOST" default:"localhost"`
	Port            int           `env:"DB_PORT" default:"5432"`
	Database        string        `env:"DB_DATABASE"`
	Username        string        `env:"DB_USERNAME"`
	Password        string        `env:"DB_PASSWORD"`
	SSLMode         string        `env:"DB_SSLMODE" default:"disable"`
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" default:"5m"`
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" default:"1m"`
}

// Open opens a database connection with the given configuration.
func Open(cfg Config) (*DB, error) {
	dsn, err := BuildDSN(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	db, err := sql.Open(cfg.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db, driver: cfg.Driver}, nil
}

// OpenWithDSN opens a database connection with a DSN string.
func OpenWithDSN(driver, dsn string) (*DB, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db, driver: driver}, nil
}

// BuildDSN builds a DSN string from the configuration.
func BuildDSN(cfg Config) (string, error) {
	switch cfg.Driver {
	case "postgres", "postgresql":
		return buildPostgresDSN(cfg), nil
	case "mysql":
		return buildMySQLDSN(cfg), nil
	case "sqlite3", "sqlite":
		return cfg.Database, nil
	default:
		return "", fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}
}

// buildPostgresDSN builds a PostgreSQL DSN.
func buildPostgresDSN(cfg Config) string {
	var parts []string

	if cfg.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", cfg.Host))
	}
	if cfg.Port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", cfg.Port))
	}
	if cfg.Username != "" {
		parts = append(parts, fmt.Sprintf("user=%s", cfg.Username))
	}
	if cfg.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", cfg.Password))
	}
	if cfg.Database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", cfg.Database))
	}
	if cfg.SSLMode != "" {
		parts = append(parts, fmt.Sprintf("sslmode=%s", cfg.SSLMode))
	}

	return strings.Join(parts, " ")
}

// buildMySQLDSN builds a MySQL DSN.
func buildMySQLDSN(cfg Config) string {
	// Format: user:password@tcp(host:port)/database?params
	u := &url.URL{
		User: url.UserPassword(cfg.Username, cfg.Password),
		Host: fmt.Sprintf("tcp(%s:%d)", cfg.Host, cfg.Port),
		Path: "/" + cfg.Database,
	}

	q := u.Query()
	q.Set("parseTime", "true")
	u.RawQuery = q.Encode()

	// MySQL DSN format is different
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
}

// Driver returns the database driver name.
func (db *DB) Driver() string {
	return db.driver
}

// HealthCheck checks if the database is reachable.
func (db *DB) HealthCheck(ctx context.Context) error {
	return db.PingContext(ctx)
}

// Stats returns database statistics.
func (db *DB) Stats() sql.DBStats {
	return db.DB.Stats()
}

// Querier is an interface for executing queries.
// It's implemented by both *sql.DB and *sql.Tx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// Ensure DB implements Querier
var _ Querier = (*sql.DB)(nil)
var _ Querier = (*sql.Tx)(nil)
