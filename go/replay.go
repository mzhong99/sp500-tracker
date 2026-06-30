package main

import (
	"fmt"
	"sort"
	"time"
)

func PrintMembers(members []Constituent, targetDate time.Time) {
	fmt.Printf("S&P 500 members on %s\n", targetDate.Format("2006-01-02"))
	fmt.Printf("count: %d\n\n", len(members))

	for _, c := range members {
		fmt.Printf("%-6s %s\n", c.Symbol, c.Security)
	}
}

func ReplayMembers(current []Constituent, changes []Change, target time.Time) []Constituent {
	members := make(map[string]Constituent)

	for _, c := range current {
		members[c.Symbol] = c
	}

	for _, change := range changes {
		if !change.Date.After(target) {
			break
		}

		if change.AddedSymbol != "" {
			delete(members, change.AddedSymbol)
		}

		if change.RemovedSymbol != "" {
			members[change.RemovedSymbol] = Constituent{
				Symbol:      change.RemovedSymbol,
				Security:    change.RemovedCompany,
				Sector:      "",
				SubIndustry: "",
			}
		}
	}

	out := make([]Constituent, 0, len(members))
	for _, c := range members {
		out = append(out, c)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Symbol < out[j].Symbol
	})

	return out
}
