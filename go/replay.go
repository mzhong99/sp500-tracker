package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

func ReplayMembers(current []Constituent, changes []Change, targetDate time.Time) []Constituent {
	members := make(map[string]Constituent)

	for _, c := range current {
		members[c.Symbol] = c
	}

	for _, ch := range changes {
		changeDate, err := parseWikiDate(ch.Date)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping change with bad date %q: %v\n", ch.Date, err)
			continue
		}

		if !changeDate.After(targetDate) {
			break
		}

		if ch.AddedSymbol != "" {
			delete(members, ch.AddedSymbol)
		}

		if ch.RemovedSymbol != "" {
			members[ch.RemovedSymbol] = Constituent{
				Symbol:   ch.RemovedSymbol,
				Security: ch.RemovedCompany,
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

func PrintMembers(members []Constituent, targetDate time.Time) {
	fmt.Printf("S&P 500 members on %s\n", targetDate.Format("2006-01-02"))
	fmt.Printf("count: %d\n\n", len(members))

	for _, c := range members {
		fmt.Printf("%-6s %s\n", c.Symbol, c.Security)
	}
}

func parseWikiDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	layouts := []string{
		"January 2, 2006",
		"Jan 2, 2006",
		"2006-01-02",
	}

	var lastErr error
	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, lastErr
}
