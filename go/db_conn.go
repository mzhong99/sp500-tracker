package main

import (
	"database/sql"
	"os"
	"strings"
	"time"
)

func dbConnect() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://sp500:sp500@postgres:5432/sp500?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func normalizeSymbol(symbol string) string {
	symbol = strings.TrimSpace(symbol)

	if i := strings.Index(symbol, "<"); i >= 0 {
		symbol = strings.TrimSpace(symbol[:i])
	}

	return strings.ToUpper(symbol)
}
