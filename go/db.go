package main

import (
	"database/sql"
	"fmt"
	"log"
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

func dbCurrentConstituents() ([]Constituent, error) {
	db, err := dbConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
SELECT symbol, security, sector, sub_industry
FROM current_constituents
ORDER BY symbol
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Constituent

	for rows.Next() {
		var c Constituent
		var sector, subIndustry sql.NullString

		if err := rows.Scan(
			&c.Symbol,
			&c.Security,
			&sector,
			&subIndustry,
		); err != nil {
			return nil, err
		}

		c.Sector = sector.String
		c.SubIndustry = subIndustry.String

		out = append(out, c)
	}

	return out, rows.Err()
}

func dbChangeEvents() ([]Change, error) {
	db, err := dbConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
SELECT change_date, added_symbol, added_company, removed_symbol, removed_company, reason
FROM change_events
ORDER BY change_date DESC, id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Change

	for rows.Next() {
		var ch Change
		var addedSymbol, addedCompany sql.NullString
		var removedSymbol, removedCompany sql.NullString
		var reason sql.NullString

		if err := rows.Scan(
			&ch.Date,
			&addedSymbol,
			&addedCompany,
			&removedSymbol,
			&removedCompany,
			&reason,
		); err != nil {
			return nil, err
		}

		ch.AddedSymbol = addedSymbol.String
		ch.AddedCompany = addedCompany.String
		ch.RemovedSymbol = removedSymbol.String
		ch.RemovedCompany = removedCompany.String
		ch.Reason = reason.String

		out = append(out, ch)
	}

	return out, rows.Err()
}

func dbCurrent() {
	current, err := dbCurrentConstituents()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded current constituents from database: %d\n", len(current))

	if len(current) > 0 {
		fmt.Println("First:")
		fmt.Println(current[0].Symbol)
		fmt.Println(current[0].Security)
		fmt.Println(current[0].Sector)
		fmt.Println(current[0].SubIndustry)
	}
}

func dbChanges() {
	changes, err := dbChangeEvents()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded constituent changes from database: %d\n", len(changes))

	limit := 5
	if len(changes) < limit {
		limit = len(changes)
	}

	for i := 0; i < limit; i++ {
		fmt.Printf("\n%d.\n", i+1)
		fmt.Println(changes[i].Date.Format("January 2, 2006"))
		fmt.Printf("Added: %s %s\n", changes[i].AddedSymbol, changes[i].AddedCompany)
		fmt.Printf("Removed: %s %s\n", changes[i].RemovedSymbol, changes[i].RemovedCompany)
		fmt.Printf("Reason: %s\n", changes[i].Reason)
	}
}

func dbMembers(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: sp500 db members YYYY-MM-DD")
		os.Exit(2)
	}

	target, err := time.Parse("2006-01-02", args[0])
	if err != nil {
		log.Fatal(err)
	}

	current, err := dbCurrentConstituents()
	if err != nil {
		log.Fatal(err)
	}

	changes, err := dbChangeEvents()
	if err != nil {
		log.Fatal(err)
	}

	members := ReplayMembers(current, changes, target)

	for _, m := range members {
		fmt.Printf("%-6s %s\n", m.Symbol, m.Security)
	}
}
