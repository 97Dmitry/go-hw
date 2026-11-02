package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func DSN() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	u := url.URL{Scheme: "postgres", Host: net.JoinHostPort(Env("DB_HOST", "localhost"), Env("DB_PORT", "5432")), User: url.UserPassword(Env("DB_USER", "postgres"), Env("DB_PASS", "postgres")), Path: "/" + Env("DB_NAME", "cashapp"), RawQuery: "sslmode=disable"}
	return u.String()
}

func Open(ctx context.Context) (*sql.DB, error) {
	conn, err := sql.Open("pgx", DSN())
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err = conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("connect PostgreSQL (check DATABASE_URL or DB_*): %w", err)
	}
	return conn, nil
}
