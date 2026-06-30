package main

import (
	"fmt"
	"sort"
)

type CoverageAudit struct {
	Provider          string
	HistoricalSymbols int
	ProviderSymbols   int
	Matched           int
	Missing           int
	Coverage          float64
	MissingSymbols    []string
}

func NewCoverageAudit(provider string, historicalSymbols, providerSymbols map[string]bool) CoverageAudit {
	var missing []string
	matched := 0

	for symbol := range historicalSymbols {
		if providerSymbols[symbol] {
			matched++
		} else {
			missing = append(missing, symbol)
		}
	}

	sort.Strings(missing)

	coverage := 0.0
	if len(historicalSymbols) > 0 {
		coverage = float64(matched) / float64(len(historicalSymbols)) * 100.0
	}

	return CoverageAudit{
		Provider:          provider,
		HistoricalSymbols: len(historicalSymbols),
		ProviderSymbols:   len(providerSymbols),
		Matched:           matched,
		Missing:           len(missing),
		Coverage:          coverage,
		MissingSymbols:    missing,
	}
}

func PrintCoverageAudit(a CoverageAudit) {
	fmt.Printf("%s coverage audit\n", a.Provider)
	fmt.Println()
	fmt.Printf("Historical symbols: %d\n", a.HistoricalSymbols)
	fmt.Printf("Provider symbols:   %d\n", a.ProviderSymbols)
	fmt.Printf("Matched:            %d\n", a.Matched)
	fmt.Printf("Missing:            %d\n", a.Missing)
	fmt.Printf("Coverage:           %.1f%%\n", a.Coverage)

	if len(a.MissingSymbols) > 0 {
		fmt.Println()
		fmt.Println("Missing symbols:")
		for _, symbol := range a.MissingSymbols {
			fmt.Println(symbol)
		}
	}
}
