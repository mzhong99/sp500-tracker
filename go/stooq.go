package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type DailyPrice struct {
	Symbol string
	Date   time.Time

	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

func handleStooqCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sp500 stooq <command>")
	}

	switch args[0] {
	case "ingest":
		return handleStooqIngest(args)

	default:
		return fmt.Errorf("unknown stooq command: %s", args[0])
	}
}

func handleStooqIngest(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: sp500 stooq ingest FILE")
	}

	prices, err := readStooqDailyPriceFile(args[1])
	if err != nil {
		return err
	}

	fmt.Printf("Parsed %d daily prices\n", len(prices))

	if len(prices) == 0 {
		return nil
	}

	if err := insertDailyPrices(prices, "stooq"); err != nil {
		return err
	}

	fmt.Printf("Inserted %d daily_prices rows for %s\n", len(prices), prices[0].Symbol)

	return nil
}

func readStooqDailyPriceFile(path string) ([]DailyPrice, error) {
	symbol := symbolFromStooqPath(path)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseStooqDailyPrices(symbol, f)
}

func symbolFromStooqPath(path string) string {
	base := path
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}

	base = strings.TrimSuffix(base, ".txt")
	base = strings.TrimSuffix(base, ".csv")
	base = strings.TrimSuffix(base, ".us")
	base = strings.TrimSuffix(base, ".US")

	symbol := strings.ToUpper(base)
	symbol = strings.ReplaceAll(symbol, "-", ".")

	return symbol
}

func parseStooqDailyPrices(symbol string, r io.Reader) ([]DailyPrice, error) {
	reader := csv.NewReader(r)

	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	expected := []string{
		"<TICKER>",
		"<PER>",
		"<DATE>",
		"<TIME>",
		"<OPEN>",
		"<HIGH>",
		"<LOW>",
		"<CLOSE>",
		"<VOL>",
		"<OPENINT>",
	}

	if len(header) != len(expected) {
		return nil, fmt.Errorf("unexpected stooq header: %v", header)
	}

	for i := range expected {
		if header[i] != expected[i] {
			return nil, fmt.Errorf("unexpected stooq header: %v", header)
		}
	}

	var prices []DailyPrice

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		date, err := time.Parse("20060102", row[2])
		if err != nil {
			return nil, err
		}

		open, err := strconv.ParseFloat(row[4], 64)
		if err != nil {
			return nil, err
		}

		high, err := strconv.ParseFloat(row[5], 64)
		if err != nil {
			return nil, err
		}

		low, err := strconv.ParseFloat(row[6], 64)
		if err != nil {
			return nil, err
		}

		closePrice, err := strconv.ParseFloat(row[7], 64)
		if err != nil {
			return nil, err
		}

		volume, err := strconv.ParseInt(row[8], 10, 64)
		if err != nil {
			return nil, err
		}

		prices = append(prices, DailyPrice{
			Symbol: symbol,
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closePrice,
			Volume: volume,
		})
	}

	return prices, nil
}
