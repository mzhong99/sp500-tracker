package main

import (
	"database/sql"
)

func dbCurrentConstituents() ([]Constituent, error) {
	db, err := dbConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
SELECT symbol, security, sector, sub_industry
FROM current_constituents
ORDER BY symbol
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Constituent

	for rows.Next() {
		var c Constituent
		var sector, subIndustry sql.NullString

		if err := rows.Scan(
			&c.Symbol,
			&c.Security,
			&sector,
			&subIndustry,
		); err != nil {
			return nil, err
		}

		c.Sector = sector.String
		c.SubIndustry = subIndustry.String

		out = append(out, c)
	}

	return out, rows.Err()
}

func dbChangeEvents() ([]Change, error) {
	db, err := dbConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
SELECT change_date, added_symbol, added_company, removed_symbol, removed_company, reason
FROM change_events
ORDER BY change_date DESC, id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Change

	for rows.Next() {
		var ch Change
		var addedSymbol, addedCompany sql.NullString
		var removedSymbol, removedCompany sql.NullString
		var reason sql.NullString

		if err := rows.Scan(
			&ch.Date,
			&addedSymbol,
			&addedCompany,
			&removedSymbol,
			&removedCompany,
			&reason,
		); err != nil {
			return nil, err
		}

		ch.AddedSymbol = addedSymbol.String
		ch.AddedCompany = addedCompany.String
		ch.RemovedSymbol = removedSymbol.String
		ch.RemovedCompany = removedCompany.String
		ch.Reason = reason.String

		out = append(out, ch)
	}

	return out, rows.Err()
}

func loadHistoricalSP500Symbols() (map[string]string, error) {
	db, err := dbConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT symbol, max(name)
		FROM (
			SELECT symbol, security AS name
			FROM current_constituents

			UNION ALL

			SELECT added_symbol AS symbol, added_company AS name
			FROM change_events
			WHERE added_symbol IS NOT NULL

			UNION ALL

			SELECT removed_symbol AS symbol, removed_company AS name
			FROM change_events
			WHERE removed_symbol IS NOT NULL
		) t
		WHERE symbol IS NOT NULL
		  AND symbol <> ''
		GROUP BY symbol
		ORDER BY symbol
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	symbols := make(map[string]string)

	for rows.Next() {
		var symbol string
		var name sql.NullString

		if err := rows.Scan(&symbol, &name); err != nil {
			return nil, err
		}

		symbol = normalizeSymbol(symbol)
		if symbol == "" {
			continue
		}

		if name.Valid {
			symbols[symbol] = name.String
		} else {
			symbols[symbol] = ""
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return symbols, nil
}
