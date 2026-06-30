package main

import (
	"database/sql"
	"fmt"
	"time"
)

func dbPrices(symbol string) error {
	db, err := dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	var count int
	var minDate time.Time
	var maxDate time.Time

	err = db.QueryRow(`
		SELECT
			count(*),
			min(price_date),
			max(price_date)
		FROM daily_prices
		WHERE symbol = $1
		  AND source = 'stooq'
	`, symbol).Scan(&count, &minDate, &maxDate)

	if err != nil {
		return err
	}

	if count == 0 {
		fmt.Printf("No daily_prices rows for %s\n", symbol)
		return nil
	}

	fmt.Printf("%s daily_prices\n", symbol)
	fmt.Printf("rows:  %d\n", count)
	fmt.Printf("range: %s -> %s\n\n",
		minDate.Format("2006-01-02"),
		maxDate.Format("2006-01-02"),
	)

	fmt.Println("first:")

	if err := printPriceRows(db, symbol, "ASC"); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("last:")

	if err := printPriceRows(db, symbol, "DESC"); err != nil {
		return err
	}

	return nil
}

func printPriceRows(db *sql.DB, symbol string, order string) error {
	query := fmt.Sprintf(`
		SELECT price_date, close, volume
		FROM daily_prices
		WHERE symbol = $1
		  AND source = 'stooq'
		ORDER BY price_date %s
		LIMIT 5
	`, order)

	rows, err := db.Query(query, symbol)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var date time.Time
		var close float64
		var volume int64

		if err := rows.Scan(&date, &close, &volume); err != nil {
			return err
		}

		fmt.Printf("%s close=%8.2f volume=%d\n",
			date.Format("2006-01-02"),
			close,
			volume,
		)
	}

	return rows.Err()
}
