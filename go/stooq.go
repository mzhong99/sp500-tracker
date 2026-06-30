package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func handleStooqCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sp500 stooq <command>")
	}

	switch args[0] {
	case "ingest":
		return handleStooqIngest(args)

	case "ingest-dir":
		return handleStooqIngestDir(args)
	case "audit":
		return handleStooqAudit(args)

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
	if err == io.EOF {
		return nil, nil
	}
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

		volumeFloat, err := strconv.ParseFloat(row[8], 64)
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
			Volume: volumeFloat,
		})
	}

	return prices, nil
}

func handleStooqIngestDir(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: sp500 stooq ingest-dir DIR")
	}

	root := args[1]

	var files []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		name := strings.ToLower(d.Name())
		if strings.HasSuffix(name, ".us.txt") {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("Found %d Stooq .us.txt files\n", len(files))

	totalRows := 0

	for i, file := range files {
		prices, err := readStooqDailyPriceFile(file)
		if err != nil {
			return fmt.Errorf("read %s: %w", file, err)
		}

		if len(prices) == 0 {
			fmt.Printf("[%d/%d] %s skipped: no rows\n", i+1, len(files), file)
			continue
		}

		if err := insertDailyPrices(prices, "stooq"); err != nil {
			return fmt.Errorf("insert %s: %w", file, err)
		}

		totalRows += len(prices)

		fmt.Printf(
			"[%d/%d] %-8s inserted %d rows\n",
			i+1,
			len(files),
			prices[0].Symbol,
			len(prices),
		)
	}

	fmt.Printf("\nInserted %d total daily_prices rows\n", totalRows)

	return nil
}

func handleStooqAudit(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: sp500 stooq audit DIR")
	}

	root := args[1]

	spSymbols, err := loadHistoricalSP500Symbols()
	if err != nil {
		return err
	}

	stooqSymbols := map[string]struct{}{}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		name := strings.ToLower(d.Name())
		if strings.HasSuffix(name, ".us.txt") {
			symbol := symbolFromStooqPath(path)
			stooqSymbols[symbol] = struct{}{}
		}

		return nil
	})
	if err != nil {
		return err
	}

	var missing []string
	matched := 0

	for symbol := range spSymbols {
		if _, ok := stooqSymbols[symbol]; ok {
			matched++
		} else {
			missing = append(missing, symbol)
		}
	}

	sort.Strings(missing)

	coverage := 0.0
	if len(spSymbols) > 0 {
		coverage = float64(matched) / float64(len(spSymbols)) * 100
	}

	fmt.Printf("Historical S&P symbols: %d\n", len(spSymbols))
	fmt.Printf("Stooq symbols:          %d\n", len(stooqSymbols))
	fmt.Printf("Matched:                %d\n", matched)
	fmt.Printf("Missing:                %d\n", len(missing))
	fmt.Printf("Coverage:               %.2f%%\n\n", coverage)

	fmt.Println("Missing symbols:")
	for _, symbol := range missing {
		name := spSymbols[symbol]
		if name == "" {
			fmt.Println(symbol)
		} else {
			fmt.Printf("%-8s %s\n", symbol, name)
		}
	}

	return nil
}
