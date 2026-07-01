package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func NewMySQL(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(5 * time.Minute)

	if err := pingWithRetry(conn, 30, time.Second); err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping mysql: %w; close mysql: %v", err, closeErr)
		}
		return nil, fmt.Errorf("ping mysql: %w", err)
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
