package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
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

func dbPing() error {
	db, err := dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	fmt.Println("connected to postgres")
	return nil
}

func dbInit() error {
	db, err := dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	schema := `
CREATE TABLE IF NOT EXISTS current_constituents (
    symbol TEXT PRIMARY KEY,
    security TEXT NOT NULL,
    sector TEXT,
    sub_industry TEXT,
    loaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS change_events (
    id BIGSERIAL PRIMARY KEY,
    change_date DATE NOT NULL,
    added_symbol TEXT,
    added_company TEXT,
    removed_symbol TEXT,
    removed_company TEXT,
    reason TEXT,
    loaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_change_events_date
ON change_events(change_date);
`
	_, err = db.Exec(schema)
	return err
}

func dbReplaceCurrentConstituents(db *sql.DB, rows []Constituent) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`TRUNCATE current_constituents`); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
INSERT INTO current_constituents (
    symbol,
    security,
    sector,
    sub_industry
) VALUES ($1, $2, $3, $4)
`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range rows {
		if _, err := stmt.Exec(
			r.Symbol,
			r.Security,
			nullableString(r.Sector),
			nullableString(r.SubIndustry),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func dbReplaceChangeEvents(db *sql.DB, rows []Change) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`TRUNCATE change_events RESTART IDENTITY`); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
INSERT INTO change_events (
    change_date,
    added_symbol,
    added_company,
    removed_symbol,
    removed_company,
    reason
) VALUES ($1, $2, $3, $4, $5, $6)
`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range rows {
		if _, err := stmt.Exec(
			dateOnly(r.Date),
			nullableString(r.AddedSymbol),
			nullableString(r.AddedCompany),
			nullableString(r.RemovedSymbol),
			nullableString(r.RemovedCompany),
			nullableString(r.Reason),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
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

func wikiIngest() error {
	current, changes := loadWikiData()

	db, err := dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	if err := dbInit(); err != nil {
		return err
	}

	if err := dbReplaceCurrentConstituents(db, current); err != nil {
		return err
	}

	if err := dbReplaceChangeEvents(db, changes); err != nil {
		return err
	}

	fmt.Printf("Inserted current constituents: %d\n", len(current))
	fmt.Printf("Inserted change events: %d\n", len(changes))

	return nil
}
