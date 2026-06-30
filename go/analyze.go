package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

type ReturnReport struct {
	Symbol     string
	First      DailyPrice
	Last       DailyPrice
	TradingDay int

	PriceMultiple float64
	TotalReturn   float64
	CAGR          float64

	Drawdown   DrawdownResult
	DailyStats DailyReturnStats
}

type DrawdownResult struct {
	MaxDrawdown float64
	PeakDate    time.Time
	PeakClose   float64
	TroughDate  time.Time
	TroughClose float64
}

type DailyReturnStats struct {
	Count                int
	Mean                 float64
	Median               float64
	WorstReturn          float64
	WorstDate            time.Time
	BestReturn           float64
	BestDate             time.Time
	AnnualizedVolatility float64
}

func handleAnalyze(db *sql.DB, args []string) {
	if len(args) < 1 {
		fmt.Println("usage: sp500 analyze <command>")
		fmt.Println()
		fmt.Println("commands:")
		fmt.Println("  returns SYMBOL")
		fmt.Println("  drawdown SYMBOL")
		fmt.Println("  volatility SYMBOL")
		os.Exit(1)
	}

	switch args[0] {
	case "returns":
		if len(args) != 2 {
			fmt.Println("usage: sp500 analyze returns SYMBOL")
			os.Exit(1)
		}

		if err := AnalyzeReturns(db, args[1], "stooq"); err != nil {
			log.Fatal(err)
		}

	case "drawdown":
		if len(args) != 2 {
			fmt.Println("usage: sp500 analyze drawdown SYMBOL")
			os.Exit(1)
		}

		if err := AnalyzeDrawdown(db, args[1], "stooq"); err != nil {
			log.Fatal(err)
		}

	case "volatility":
		if len(args) != 2 {
			fmt.Println("usage: sp500 analyze volatility SYMBOL")
			os.Exit(1)
		}

		if err := AnalyzeVolatility(db, args[1], "stooq"); err != nil {
			log.Fatal(err)
		}

	case "compare":
		if len(args) < 3 {
			fmt.Println("usage: sp500 analyze compare SYMBOL [SYMBOL...]")
			os.Exit(1)
		}

		if err := AnalyzeCompare(db, args[1:], "stooq"); err != nil {
			log.Fatal(err)
		}

	default:
		fmt.Printf("unknown analyze command: %s\n", args[0])
		os.Exit(1)
	}
}

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

func BuildReturnReport(symbol string, prices []DailyPrice) (ReturnReport, error) {
	if len(prices) < 2 {
		return ReturnReport{}, fmt.Errorf("not enough price data for %s", symbol)
	}

	first := prices[0]
	last := prices[len(prices)-1]

	if first.Close <= 0 || last.Close <= 0 {
		return ReturnReport{}, fmt.Errorf("invalid close price for %s", symbol)
	}

	days := last.Date.Sub(first.Date).Hours() / 24.0
	years := days / 365.25

	if years <= 0 {
		return ReturnReport{}, fmt.Errorf("invalid date range for %s", symbol)
	}

	multiple := last.Close / first.Close
	totalReturn := multiple - 1.0
	cagr := math.Pow(multiple, 1.0/years) - 1.0

	drawdown, err := ComputeMaxDrawdown(prices)
	if err != nil {
		return ReturnReport{}, err
	}

	dailyStats, err := ComputeDailyReturnStats(prices)
	if err != nil {
		return ReturnReport{}, err
	}

	return ReturnReport{
		Symbol:        symbol,
		First:         first,
		Last:          last,
		TradingDay:    len(prices),
		PriceMultiple: multiple,
		TotalReturn:   totalReturn,
		CAGR:          cagr,
		Drawdown:      drawdown,
		DailyStats:    dailyStats,
	}, nil
}

func PrintReturnReport(report ReturnReport) {
	fmt.Println(report.Symbol)
	fmt.Println()
	fmt.Println("History")
	fmt.Println("-------")
	fmt.Printf("%s → %s\n", FormatDate(report.First.Date), FormatDate(report.Last.Date))
	fmt.Printf("%d trading days\n", report.TradingDay)

	fmt.Println()
	fmt.Println("Prices")
	fmt.Println("------")
	fmt.Printf("First close:         %.4f\n", report.First.Close)
	fmt.Printf("Last close:          %.4f\n", report.Last.Close)

	fmt.Println()
	fmt.Println("Return")
	fmt.Println("------")
	fmt.Printf("Price multiple:      %.2fx\n", report.PriceMultiple)
	fmt.Printf("Total price return:  %.2f%%\n", report.TotalReturn*100.0)
	fmt.Printf("CAGR:                %.2f%%\n", report.CAGR*100.0)

	fmt.Println()
	fmt.Println("Drawdown")
	fmt.Println("--------")
	fmt.Printf("Maximum drawdown:    %.2f%%\n", report.Drawdown.MaxDrawdown*100.0)
	fmt.Printf("Peak:                %s close %.4f\n", FormatDate(report.Drawdown.PeakDate), report.Drawdown.PeakClose)
	fmt.Printf("Trough:              %s close %.4f\n", FormatDate(report.Drawdown.TroughDate), report.Drawdown.TroughClose)

	fmt.Println()
	fmt.Println("Daily returns")
	fmt.Println("-------------")
	fmt.Printf("Return days:          %d\n", report.DailyStats.Count)
	fmt.Printf("Mean daily return:    %.4f%%\n", report.DailyStats.Mean*100.0)
	fmt.Printf("Median daily return:  %.4f%%\n", report.DailyStats.Median*100.0)
	fmt.Printf("Worst day:            %.2f%% on %s\n", report.DailyStats.WorstReturn*100.0, FormatDate(report.DailyStats.WorstDate))
	fmt.Printf("Best day:             %.2f%% on %s\n", report.DailyStats.BestReturn*100.0, FormatDate(report.DailyStats.BestDate))
	fmt.Printf("Annualized volatility: %.2f%%\n", report.DailyStats.AnnualizedVolatility*100.0)

	fmt.Println()
	fmt.Println("Note: price return only; dividends are not included.")
}

func ComputeMaxDrawdown(prices []DailyPrice) (DrawdownResult, error) {
	if len(prices) < 2 {
		return DrawdownResult{}, fmt.Errorf("not enough prices to compute drawdown")
	}

	peak := prices[0]
	maxDrawdown := 0.0

	result := DrawdownResult{
		MaxDrawdown: 0.0,
		PeakDate:    prices[0].Date,
		PeakClose:   prices[0].Close,
		TroughDate:  prices[0].Date,
		TroughClose: prices[0].Close,
	}

	for _, p := range prices {
		if p.Close <= 0 {
			return DrawdownResult{}, fmt.Errorf("invalid close price on %s", FormatDate(p.Date))
		}

		if p.Close > peak.Close {
			peak = p
		}

		drawdown := (p.Close / peak.Close) - 1.0

		if drawdown < maxDrawdown {
			maxDrawdown = drawdown

			result = DrawdownResult{
				MaxDrawdown: drawdown,
				PeakDate:    peak.Date,
				PeakClose:   peak.Close,
				TroughDate:  p.Date,
				TroughClose: p.Close,
			}
		}
	}

	return result, nil
}

func ComputeDailyReturnStats(prices []DailyPrice) (DailyReturnStats, error) {
	if len(prices) < 2 {
		return DailyReturnStats{}, fmt.Errorf("not enough prices to compute daily returns")
	}

	returns := make([]float64, 0, len(prices)-1)

	stats := DailyReturnStats{
		WorstReturn: 0.0,
		BestReturn:  0.0,
	}

	for i := 1; i < len(prices); i++ {
		prev := prices[i-1]
		curr := prices[i]

		if prev.Close <= 0 || curr.Close <= 0 {
			return DailyReturnStats{}, fmt.Errorf("invalid close price near %s", FormatDate(curr.Date))
		}

		r := (curr.Close / prev.Close) - 1.0
		returns = append(returns, r)

		if i == 1 || r < stats.WorstReturn {
			stats.WorstReturn = r
			stats.WorstDate = curr.Date
		}

		if i == 1 || r > stats.BestReturn {
			stats.BestReturn = r
			stats.BestDate = curr.Date
		}
	}

	stats.Count = len(returns)
	stats.Mean = meanFloat64(returns)
	stats.Median = medianFloat64(returns)
	stats.AnnualizedVolatility = sampleStdDevFloat64(returns) * math.Sqrt(252.0)

	return stats, nil
}

func meanFloat64(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}

	var sum float64
	for _, x := range xs {
		sum += x
	}

	return sum / float64(len(xs))
}

func medianFloat64(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}

	ys := append([]float64(nil), xs...)
	sort.Float64s(ys)

	n := len(ys)
	if n%2 == 1 {
		return ys[n/2]
	}

	return (ys[n/2-1] + ys[n/2]) / 2.0
}

func sampleStdDevFloat64(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}

	mean := meanFloat64(xs)

	var sumSquares float64
	for _, x := range xs {
		diff := x - mean
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(len(xs)-1)
	return math.Sqrt(variance)
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
