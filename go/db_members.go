package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

func dbPing() error {
	db, err := dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	fmt.Println("connected to postgres")
	return nil
}

func dbCurrent() {
	current, err := dbCurrentConstituents()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded current constituents from database: %d\n", len(current))

	if len(current) > 0 {
		fmt.Println("First:")
		fmt.Println(current[0].Symbol)
		fmt.Println(current[0].Security)
		fmt.Println(current[0].Sector)
		fmt.Println(current[0].SubIndustry)
	}
}

func dbChanges() {
	changes, err := dbChangeEvents()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded constituent changes from database: %d\n", len(changes))

	limit := 5
	if len(changes) < limit {
		limit = len(changes)
	}

	for i := 0; i < limit; i++ {
		fmt.Printf("\n%d.\n", i+1)
		fmt.Println(changes[i].Date.Format("January 2, 2006"))
		fmt.Printf("Added: %s %s\n", changes[i].AddedSymbol, changes[i].AddedCompany)
		fmt.Printf("Removed: %s %s\n", changes[i].RemovedSymbol, changes[i].RemovedCompany)
		fmt.Printf("Reason: %s\n", changes[i].Reason)
	}
}

func dbMembers(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: sp500 db members YYYY-MM-DD")
		os.Exit(2)
	}

	target, err := time.Parse("2006-01-02", args[0])
	if err != nil {
		log.Fatal(err)
	}

	current, err := dbCurrentConstituents()
	if err != nil {
		log.Fatal(err)
	}

	changes, err := dbChangeEvents()
	if err != nil {
		log.Fatal(err)
	}

	members := ReplayMembers(current, changes, target)

	for _, m := range members {
		fmt.Printf("%-6s %s\n", m.Symbol, m.Security)
	}
}

func dbSymbol(symbol string) error {
	db, err := dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	var security, sector, subIndustry string

	err = db.QueryRow(`
		SELECT security, sector, sub_industry
		FROM current_constituents
		WHERE symbol = $1
	`, symbol).Scan(&security, &sector, &subIndustry)

	if err == sql.ErrNoRows {
		fmt.Printf("%s is not in current_constituents\n\n", symbol)
	} else if err != nil {
		return err
	} else {
		fmt.Printf("%s is currently in the S&P 500\n", symbol)
		fmt.Printf("Security:     %s\n", security)
		fmt.Printf("Sector:       %s\n", sector)
		fmt.Printf("Sub-industry: %s\n\n", subIndustry)
	}

	rows, err := db.Query(`
		SELECT
			change_date,
			added_symbol,
			added_company,
			removed_symbol,
			removed_company,
			reason
		FROM change_events
		WHERE added_symbol = $1
		   OR removed_symbol = $1
		ORDER BY change_date DESC
	`, symbol)
	if err != nil {
		return err
	}
	defer rows.Close()

	found := false

	fmt.Println("Change events:")

	for rows.Next() {
		found = true

		var change Change

		if err := rows.Scan(
			&change.Date,
			&change.AddedSymbol,
			&change.AddedCompany,
			&change.RemovedSymbol,
			&change.RemovedCompany,
			&change.Reason,
		); err != nil {
			return err
		}

		fmt.Printf("%s\n", change.Date.Format("2006-01-02"))

		if change.AddedSymbol == symbol {
			fmt.Printf("  Added:   %s %s\n", change.AddedSymbol, change.AddedCompany)
		}

		if change.RemovedSymbol == symbol {
			fmt.Printf("  Removed: %s %s\n", change.RemovedSymbol, change.RemovedCompany)
		}

		if change.Reason != "" {
			fmt.Printf("  Reason:  %s\n", change.Reason)
		}

		fmt.Println()
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if !found {
		fmt.Println("  none")
	}

	return nil
}
