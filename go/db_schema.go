package main

func dbInit() error {
	db, err := dbConnect()
	if err != nil {
		return err
	}
	defer db.Close()

	schema := `
CREATE TABLE IF NOT EXISTS current_constituents (
    symbol TEXT PRIMARY KEY,
    security TEXT NOT NULL,
    sector TEXT,
    sub_industry TEXT,
    loaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS change_events (
    id BIGSERIAL PRIMARY KEY,
    change_date DATE NOT NULL,
    added_symbol TEXT,
    added_company TEXT,
    removed_symbol TEXT,
    removed_company TEXT,
    reason TEXT,
    loaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS daily_prices (
    symbol TEXT NOT NULL,
    price_date DATE NOT NULL,
    open NUMERIC NOT NULL,
    high NUMERIC NOT NULL,
    low NUMERIC NOT NULL,
    close NUMERIC NOT NULL,
    volume BIGINT NOT NULL,
    source TEXT NOT NULL,
    loaded_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (symbol, price_date, source)
);

CREATE TABLE IF NOT EXISTS daily_market_caps (
    symbol TEXT NOT NULL,
    cap_date DATE NOT NULL,
    market_cap NUMERIC NOT NULL,
    source TEXT NOT NULL,
    loaded_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (symbol, cap_date, source)
);

CREATE INDEX IF NOT EXISTS idx_change_events_date
ON change_events(change_date);

CREATE INDEX IF NOT EXISTS idx_daily_market_caps_date
ON daily_market_caps(cap_date);
`
	_, err = db.Exec(schema)
	return err
}
