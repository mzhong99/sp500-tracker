package main

import (
	"database/sql"
	"time"
)

type DailyPrice struct {
	Symbol string
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
	Source string
}

func LoadDailyPrices(db *sql.DB, symbol string, source string) ([]DailyPrice, error) {
	const q = `
SELECT
    symbol,
    price_date,
    open,
    high,
    low,
    close,
    volume,
    source
FROM daily_prices
WHERE symbol = $1
  AND source = $2
ORDER BY price_date
`

	rows, err := db.Query(q, symbol, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []DailyPrice

	for rows.Next() {
		var p DailyPrice

		if err := rows.Scan(
			&p.Symbol,
			&p.Date,
			&p.Open,
			&p.High,
			&p.Low,
			&p.Close,
			&p.Volume,
			&p.Source,
		); err != nil {
			return nil, err
		}

		prices = append(prices, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return prices, nil
}

func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}
