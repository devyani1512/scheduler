package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type DB struct {
	Conn   *sql.DB
	logger *zap.Logger
}

func New(dsn string, logger *zap.Logger) (*DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(5 * time.Minute)
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("pinging db: %w", err)
	}
	logger.Info("database connected")
	return &DB{Conn: conn, logger: logger}, nil
}
func (d *DB) Close() error {
	return d.Conn.Close()
}

// scanner is a common interface for *sql.Row and *sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}
