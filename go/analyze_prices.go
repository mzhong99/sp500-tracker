package main

import (
	"database/sql"
	"fmt"
	"strings"
)

func AnalyzeReturns(db *sql.DB, symbol string, source string) error {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	prices, err := LoadDailyPrices(db, symbol, source)
	if err != nil {
		return err
	}

	report, err := BuildReturnReport(symbol, prices)
	if err != nil {
		return err
	}

	PrintReturnReport(report)
	return nil
}

func AnalyzeDrawdown(db *sql.DB, symbol string, source string) error {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	prices, err := LoadDailyPrices(db, symbol, source)
	if err != nil {
		return err
	}

	drawdown, err := BuildDrawdownReport(symbol, prices)
	if err != nil {
		return err
	}

	PrintDrawdownReport(symbol, drawdown)
	return nil
}

func BuildDrawdownReport(symbol string, prices []DailyPrice) (DrawdownResult, error) {
	if len(prices) < 2 {
		return DrawdownResult{}, fmt.Errorf("not enough price data for %s", symbol)
	}

	return ComputeMaxDrawdown(prices)
}

func PrintDrawdownReport(symbol string, drawdown DrawdownResult) {
	fmt.Println(symbol)
	fmt.Println()
	fmt.Println("Drawdown")
	fmt.Println("--------")
	fmt.Printf("Maximum drawdown:    %.2f%%\n", drawdown.MaxDrawdown*100.0)
	fmt.Printf("Peak:                %s close %.4f\n", FormatDate(drawdown.PeakDate), drawdown.PeakClose)
	fmt.Printf("Trough:              %s close %.4f\n", FormatDate(drawdown.TroughDate), drawdown.TroughClose)
	fmt.Println()
	fmt.Println("Note: price return only; dividends are not included.")
}

func AnalyzeVolatility(db *sql.DB, symbol string, source string) error {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	prices, err := LoadDailyPrices(db, symbol, source)
	if err != nil {
		return err
	}

	stats, err := BuildVolatilityReport(symbol, prices)
	if err != nil {
		return err
	}

	PrintVolatilityReport(symbol, stats)
	return nil
}

func BuildVolatilityReport(symbol string, prices []DailyPrice) (DailyReturnStats, error) {
	if len(prices) < 2 {
		return DailyReturnStats{}, fmt.Errorf("not enough price data for %s", symbol)
	}

	return ComputeDailyReturnStats(prices)
}

func PrintVolatilityReport(symbol string, stats DailyReturnStats) {
	fmt.Println(symbol)
	fmt.Println()
	fmt.Println("Volatility")
	fmt.Println("----------")
	fmt.Printf("Return days:           %d\n", stats.Count)
	fmt.Printf("Mean daily return:     %.4f%%\n", stats.Mean*100.0)
	fmt.Printf("Median daily return:   %.4f%%\n", stats.Median*100.0)
	fmt.Printf("Annualized volatility: %.2f%%\n", stats.AnnualizedVolatility*100.0)
	fmt.Printf("Worst day:             %.2f%% on %s\n", stats.WorstReturn*100.0, FormatDate(stats.WorstDate))
	fmt.Printf("Best day:              %.2f%% on %s\n", stats.BestReturn*100.0, FormatDate(stats.BestDate))
	fmt.Println()
	fmt.Println("Note: computed from daily price returns; dividends are not included.")
}

func AnalyzeCompare(db *sql.DB, symbols []string, source string) error {
	var reports []ReturnReport

	for _, rawSymbol := range symbols {
		symbol := strings.ToUpper(strings.TrimSpace(rawSymbol))
		if symbol == "" {
			continue
		}

		prices, err := LoadDailyPrices(db, symbol, source)
		if err != nil {
			return err
		}

		report, err := BuildReturnReport(symbol, prices)
		if err != nil {
			return err
		}

		reports = append(reports, report)
	}

	if len(reports) == 0 {
		return fmt.Errorf("no symbols provided")
	}

	PrintCompareReport(reports)
	return nil
}

func PrintCompareReport(reports []ReturnReport) {
	fmt.Println("Comparison")
	fmt.Println("----------")
	fmt.Println()

	fmt.Printf(
		"%-8s %-12s %-12s %10s %10s %10s %12s %8s\n",
		"Symbol",
		"First",
		"Last",
		"CAGR",
		"Vol",
		"Max DD",
		"Multiple",
		"Days",
	)

	for _, r := range reports {
		fmt.Printf(
			"%-8s %-12s %-12s %10.2f%% %10.2f%% %10.2f%% %11.2fx %8d\n",
			r.Symbol,
			FormatDate(r.First.Date),
			FormatDate(r.Last.Date),
			r.CAGR*100.0,
			r.DailyStats.AnnualizedVolatility*100.0,
			r.Drawdown.MaxDrawdown*100.0,
			r.PriceMultiple,
			r.TradingDay,
		)
	}

	fmt.Println()
	fmt.Println("Note: price returns only; dividends are not included.")
}
