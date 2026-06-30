package main

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

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

func insertDailyPrices(prices []DailyPrice, source string) error {
	if len(prices) == 0 {
		return nil
	}

	db, err := dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	symbol := prices[0].Symbol

	if _, err := tx.Exec(`
		DELETE FROM daily_prices
		WHERE symbol = $1
		  AND source = $2
	`, symbol, source); err != nil {
		return err
	}

	stmt, err := tx.Prepare(pq.CopyIn(
		"daily_prices",
		"symbol",
		"price_date",
		"open",
		"high",
		"low",
		"close",
		"volume",
		"source",
	))
	if err != nil {
		return err
	}

	for _, p := range prices {
		if p.Symbol != symbol {
			_ = stmt.Close()
			return fmt.Errorf("mixed symbols in daily price batch: %s and %s", symbol, p.Symbol)
		}

		if _, err := stmt.Exec(
			p.Symbol,
			p.Date,
			p.Open,
			p.High,
			p.Low,
			p.Close,
			p.Volume,
			source,
		); err != nil {
			_ = stmt.Close()
			return err
		}
	}

	if _, err := stmt.Exec(); err != nil {
		_ = stmt.Close()
		return err
	}

	if err := stmt.Close(); err != nil {
		return err
	}

	return tx.Commit()
}
