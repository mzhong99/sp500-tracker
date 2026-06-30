package main

import (
	"fmt"
	"sort"
)

type MissingSymbol struct {
	Symbol string
	Name   string
}

type CoverageAudit struct {
	Provider          string
	HistoricalSymbols int
	ProviderSymbols   int
	Matched           int
	Missing           int
	Coverage          float64
	MissingSymbols    []MissingSymbol
}

func NewCoverageAudit(provider string, historicalSymbols map[string]string, providerSymbols map[string]bool) CoverageAudit {
	var missing []MissingSymbol
	matched := 0

	for symbol, name := range historicalSymbols {
		if providerSymbols[symbol] {
			matched++
		} else {
			missing = append(missing, MissingSymbol{
				Symbol: symbol,
				Name:   name,
			})
		}
	}

	sort.Slice(missing, func(i, j int) bool {
		return missing[i].Symbol < missing[j].Symbol
	})

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
		for _, missing := range a.MissingSymbols {
			fmt.Printf("%-8s %s\n", missing.Symbol, missing.Name)
		}
	}
}
