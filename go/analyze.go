package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
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

	case "members":
		if err := handleAnalyzeMembers(args); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

	case "snapshot":
		if err := handleAnalyzeSnapshot(args); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

	case "missing":
		if err := handleAnalyzeMissing(args); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

	case "coverage-history":
		if err := handleAnalyzeCoverageHistory(args); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

	default:
		fmt.Printf("unknown analyze command: %s\n", args[0])
		os.Exit(1)
	}
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
