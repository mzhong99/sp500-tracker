package main

import (
	"fmt"
	"sort"
	"time"
)

type SnapshotStats struct {
	Date     time.Time
	Members  int
	Priced   int
	Missing  int
	Coverage float64

	AverageClose float64
	MedianClose  float64

	LowestSymbol string
	LowestName   string
	LowestClose  float64

	HighestSymbol string
	HighestName   string
	HighestClose  float64
}

type MemberSnapshotRow struct {
	Symbol   string
	Name     string
	Close    float64
	HasPrice bool
}

func handleAnalyzeSnapshot(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: analyze snapshot YYYY-MM-DD")
	}

	asOf, err := time.Parse("2006-01-02", args[1])
	if err != nil {
		return fmt.Errorf("invalid date %q: expected YYYY-MM-DD", args[1])
	}

	rows, err := loadMemberPriceSnapshot(asOf)
	if err != nil {
		return err
	}

	stats := computeSnapshotStats(asOf, rows)
	printSnapshotStats(stats)

	return nil
}

func loadMemberPriceSnapshot(asOf time.Time) ([]MemberSnapshotRow, error) {
	current, err := dbCurrentConstituents()
	if err != nil {
		return nil, err
	}

	changes, err := dbChangeEvents()
	if err != nil {
		return nil, err
	}

	members := ReplayMembers(current, changes, asOf)

	prices, err := loadClosesForDate(asOf)
	if err != nil {
		return nil, err
	}

	rows := make([]MemberSnapshotRow, 0, len(members))

	for _, m := range members {
		symbol := normalizeSymbol(m.Symbol)
		close, ok := prices[symbol]

		rows = append(rows, MemberSnapshotRow{
			Symbol:   symbol,
			Name:     m.Security,
			Close:    close,
			HasPrice: ok,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Symbol < rows[j].Symbol
	})

	return rows, nil
}

func computeSnapshotStats(asOf time.Time, rows []MemberSnapshotRow) SnapshotStats {
	stats := SnapshotStats{
		Date:    asOf,
		Members: len(rows),
	}

	var closes []float64

	for _, row := range rows {
		if !row.HasPrice {
			stats.Missing++
			continue
		}

		stats.Priced++
		closes = append(closes, row.Close)

		if stats.Priced == 1 || row.Close < stats.LowestClose {
			stats.LowestSymbol = row.Symbol
			stats.LowestName = row.Name
			stats.LowestClose = row.Close
		}

		if stats.Priced == 1 || row.Close > stats.HighestClose {
			stats.HighestSymbol = row.Symbol
			stats.HighestName = row.Name
			stats.HighestClose = row.Close
		}
	}

	if stats.Members > 0 {
		stats.Coverage = float64(stats.Priced) / float64(stats.Members) * 100.0
	}

	if len(closes) > 0 {
		sort.Float64s(closes)

		sum := 0.0
		for _, close := range closes {
			sum += close
		}

		stats.AverageClose = sum / float64(len(closes))

		mid := len(closes) / 2
		if len(closes)%2 == 0 {
			stats.MedianClose = (closes[mid-1] + closes[mid]) / 2.0
		} else {
			stats.MedianClose = closes[mid]
		}
	}

	return stats
}

func printSnapshotStats(stats SnapshotStats) {
	fmt.Println("Snapshot")
	fmt.Println()
	fmt.Printf("Date:               %s\n", stats.Date.Format("2006-01-02"))
	fmt.Println()
	fmt.Printf("Members:            %d\n", stats.Members)
	fmt.Printf("Priced:             %d\n", stats.Priced)
	fmt.Printf("Missing prices:     %d\n", stats.Missing)
	fmt.Printf("Coverage:           %.1f%%\n", stats.Coverage)
	fmt.Println()

	if stats.Priced == 0 {
		fmt.Println("No price data found.")
		return
	}

	fmt.Printf("Average close:      %.4f\n", stats.AverageClose)
	fmt.Printf("Median close:       %.4f\n", stats.MedianClose)
	fmt.Println()
	fmt.Println("Lowest close:")
	fmt.Printf("%-8s %-40s %12.4f\n", stats.LowestSymbol, stats.LowestName, stats.LowestClose)
	fmt.Println()
	fmt.Println("Highest close:")
	fmt.Printf("%-8s %-40s %12.4f\n", stats.HighestSymbol, stats.HighestName, stats.HighestClose)
}

func handleAnalyzeMembers(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: analyze members YYYY-MM-DD")
	}

	asOf, err := time.Parse("2006-01-02", args[1])
	if err != nil {
		return fmt.Errorf("invalid date %q: expected YYYY-MM-DD", args[1])
	}

	rows, err := loadMemberPriceSnapshot(asOf)
	if err != nil {
		return err
	}

	priced := 0
	missing := 0

	for _, row := range rows {
		if row.HasPrice {
			priced++
		} else {
			missing++
		}
	}

	fmt.Printf("Date: %s\n", asOf.Format("2006-01-02"))
	fmt.Printf("Members: %d\n", len(rows))
	fmt.Printf("Priced:  %d\n", priced)
	fmt.Printf("Missing: %d\n", missing)
	fmt.Println()
	fmt.Printf("%-8s %-40s %12s\n", "Symbol", "Company", "Close")
	fmt.Printf("%-8s %-40s %12s\n", "------", "-------", "-----")

	for _, row := range rows {
		if row.HasPrice {
			fmt.Printf("%-8s %-40s %12.4f\n", row.Symbol, row.Name, row.Close)
		} else {
			fmt.Printf("%-8s %-40s %12s\n", row.Symbol, row.Name, "MISSING")
		}
	}

	return nil
}

func loadClosesForDate(asOf time.Time) (map[string]float64, error) {
	db, err := dbConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT symbol, close
		FROM daily_prices
		WHERE price_date = $1
		  AND source = 'stooq'
	`, dateOnly(asOf))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := make(map[string]float64)

	for rows.Next() {
		var symbol string
		var close float64

		if err := rows.Scan(&symbol, &close); err != nil {
			return nil, err
		}

		prices[normalizeSymbol(symbol)] = close
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return prices, nil
}

func handleAnalyzeMissing(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: analyze missing YYYY-MM-DD")
	}

	asOf, err := time.Parse("2006-01-02", args[1])
	if err != nil {
		return fmt.Errorf("invalid date %q: expected YYYY-MM-DD", args[1])
	}

	rows, err := loadMemberPriceSnapshot(asOf)
	if err != nil {
		return err
	}

	missing := make([]MemberSnapshotRow, 0)

	for _, row := range rows {
		if !row.HasPrice {
			missing = append(missing, row)
		}
	}

	fmt.Printf("Date: %s\n", asOf.Format("2006-01-02"))
	fmt.Printf("Members: %d\n", len(rows))
	fmt.Printf("Missing prices: %d\n", len(missing))

	if len(rows) > 0 {
		coverage := float64(len(rows)-len(missing)) / float64(len(rows)) * 100.0
		fmt.Printf("Coverage: %.1f%%\n", coverage)
	}

	fmt.Println()

	if len(missing) == 0 {
		fmt.Println("No missing prices.")
		return nil
	}

	fmt.Printf("%-8s %-40s\n", "Symbol", "Company")
	fmt.Printf("%-8s %-40s\n", "------", "-------")

	for _, row := range missing {
		fmt.Printf("%-8s %-40s\n", row.Symbol, row.Name)
	}

	return nil
}

func handleAnalyzeCoverageHistory(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: analyze coverage-history")
	}

	startYear := 2000
	endYear := time.Now().Year()

	fmt.Printf("%-6s %8s %8s %8s %10s\n", "Year", "Members", "Priced", "Missing", "Coverage")
	fmt.Printf("%-6s %8s %8s %8s %10s\n", "----", "-------", "------", "-------", "--------")

	for year := startYear; year <= endYear; year++ {
		target := time.Date(year, time.January, 3, 0, 0, 0, 0, time.UTC)

		asOf, err := firstPriceDateOnOrAfter(target)
		if err != nil {
			return err
		}

		rows, err := loadMemberPriceSnapshot(asOf)
		if err != nil {
			return err
		}

		members := len(rows)
		priced := 0

		for _, row := range rows {
			if row.HasPrice {
				priced++
			}
		}

		missing := members - priced

		coverage := 0.0
		if members > 0 {
			coverage = float64(priced) / float64(members) * 100.0
		}

		fmt.Printf("%-6d %8d %8d %8d %9.1f%%\n", year, members, priced, missing, coverage)
	}

	return nil
}

func firstPriceDateOnOrAfter(asOf time.Time) (time.Time, error) {
	db, err := dbConnect()
	if err != nil {
		return time.Time{}, err
	}
	defer db.Close()

	var priceDate time.Time

	err = db.QueryRow(`
		SELECT MIN(price_date)
		FROM daily_prices
		WHERE price_date >= $1
		  AND source = 'stooq'
	`, dateOnly(asOf)).Scan(&priceDate)
	if err != nil {
		return time.Time{}, err
	}

	return priceDate, nil
}
