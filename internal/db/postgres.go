package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgres(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(5 * time.Minute)

	if err := pingWithRetry(conn, 30, time.Second); err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping postgres: %w; close postgres: %v", err, closeErr)
		}
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return conn, nil
}

func pingWithRetry(conn *sql.DB, attempts int, delay time.Duration) error {
	var err error
	for i := 0; i < attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), delay)
		err = conn.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return err
}
