package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

func handleMarketCap(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: marketcap COMMAND")
	}

	switch args[0] {
	case "ingest-csv":
		return handleMarketCapIngestCSV(args)
	default:
		return fmt.Errorf("unknown marketcap command: %s", args[0])
	}
}

func handleMarketCapIngestCSV(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: marketcap ingest-csv FILE")
	}

	filePath := args[1]

	rows, err := parseMarketCapCSV(filePath)
	if err != nil {
		return err
	}

	if err := insertMarketCaps(rows, "manual_csv"); err != nil {
		return err
	}

	fmt.Printf("Inserted market cap rows: %d\n", len(rows))
	fmt.Println("Source: manual_csv")

	return nil
}

type MarketCapRow struct {
	Symbol    string
	Date      time.Time
	MarketCap float64
}

func parseMarketCapCSV(filePath string) ([]MarketCapRow, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return nil, err
	}

	col := map[string]int{}
	for i, name := range header {
		col[name] = i
	}

	required := []string{"symbol", "cap_date", "market_cap"}
	for _, name := range required {
		if _, ok := col[name]; !ok {
			return nil, fmt.Errorf("missing required CSV column: %s", name)
		}
	}

	var rows []MarketCapRow

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		symbol := normalizeSymbol(rec[col["symbol"]])
		if symbol == "" {
			continue
		}

		capDate, err := time.Parse("2006-01-02", rec[col["cap_date"]])
		if err != nil {
			return nil, fmt.Errorf("invalid cap_date for %s: %w", symbol, err)
		}

		marketCap, err := strconv.ParseFloat(rec[col["market_cap"]], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid market_cap for %s: %w", symbol, err)
		}

		rows = append(rows, MarketCapRow{
			Symbol:    symbol,
			Date:      dateOnly(capDate),
			MarketCap: marketCap,
		})
	}

	return rows, nil
}

func insertMarketCaps(rows []MarketCapRow, source string) error {
	if len(rows) == 0 {
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

	stmt, err := tx.Prepare(`
		INSERT INTO daily_market_caps (
			symbol,
			cap_date,
			market_cap,
			source
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (symbol, cap_date, source)
		DO UPDATE SET
			market_cap = EXCLUDED.market_cap,
			loaded_at = now()
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, row := range rows {
		if _, err := stmt.Exec(
			row.Symbol,
			row.Date,
			row.MarketCap,
			source,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
