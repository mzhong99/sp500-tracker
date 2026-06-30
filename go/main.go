package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"encoding/json"
	"io"
	"sort"

	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(2)
	}

	switch {
	case os.Args[1] == "db" && os.Args[2] == "ping":
		dbPing()
	case os.Args[1] == "wiki" && os.Args[2] == "ping":
		wikiPing()
	case os.Args[1] == "wiki" && os.Args[2] == "dump":
		wikiDump()
	case os.Args[1] == "wiki" && os.Args[2] == "current":
		wikiCurrent()
	case os.Args[1] == "wiki" && os.Args[2] == "changes":
		wikiChanges()
	case os.Args[1] == "wiki" && os.Args[2] == "members":
		wikiMembers()
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  sp500ctl db ping")
	fmt.Fprintln(os.Stderr, "  sp500ctl wiki ping")
	fmt.Fprintln(os.Stderr, "  sp500ctl wiki dump")
	fmt.Fprintln(os.Stderr, "  sp500ctl wiki current")
	fmt.Fprintln(os.Stderr, "  sp500ctl wiki changes")
	fmt.Fprintln(os.Stderr, "  sp500ctl wiki members YYYY-MM-DD")
}

func dbPing() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	if err := conn.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ping failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("connected to postgres")
}

func wikiPing() {
	url := "https://en.wikipedia.org/w/api.php?action=parse&page=List_of_S%26P_500_companies&prop=wikitext&format=json"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request creation failed: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("User-Agent", "sp500-tracker/0.1")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("Wikipedia API status: %s\n", resp.Status)
}

func wikiDump() {
	url := "https://en.wikipedia.org/w/api.php?action=parse&page=List_of_S%26P_500_companies&prop=wikitext&format=json"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request creation failed: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("User-Agent", "sp500-tracker/0.1")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Wikipedia API returned: %s\n", resp.Status)
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read failed: %v\n", err)
		os.Exit(1)
	}

	var parsed struct {
		Parse struct {
			Title    string `json:"title"`
			Wikitext struct {
				Text string `json:"*"`
			} `json:"wikitext"`
		} `json:"parse"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		fmt.Fprintf(os.Stderr, "json parse failed: %v\n", err)
		os.Exit(1)
	}

	text := parsed.Parse.Wikitext.Text

	fmt.Printf("title: %s\n\n", parsed.Parse.Title)
	fmt.Print(text)
}

type Constituent struct {
	Symbol      string
	Security    string
	Sector      string
	SubIndustry string
}

func wikiCurrent() {
	text := fetchWikiText()

	table, err := extractWikiTable(text, `id="constituents"`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract table failed: %v\n", err)
		os.Exit(1)
	}

	rows := parseConstituentRows(table)

	fmt.Printf("Fetched current constituents: %d\n", len(rows))
	if len(rows) > 0 {
		fmt.Println("First:")
		fmt.Println(rows[0].Symbol)
		fmt.Println(rows[0].Security)
		fmt.Println(rows[0].Sector)
		fmt.Println(rows[0].SubIndustry)
	}
}

func fetchWikiText() string {
	url := "https://en.wikipedia.org/w/api.php?action=parse&page=List_of_S%26P_500_companies&prop=wikitext&format=json"

	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request creation failed: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("User-Agent", "sp500-tracker/0.1")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Wikipedia API returned: %s\n", resp.Status)
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read failed: %v\n", err)
		os.Exit(1)
	}

	var parsed struct {
		Parse struct {
			Title    string `json:"title"`
			Wikitext struct {
				Text string `json:"*"`
			} `json:"wikitext"`
		} `json:"parse"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		fmt.Fprintf(os.Stderr, "json parse failed: %v\n", err)
		os.Exit(1)
	}

	return parsed.Parse.Wikitext.Text
}

func extractWikiTable(text string, marker string) (string, error) {
	start := strings.Index(text, marker)
	if start == -1 {
		return "", fmt.Errorf("marker not found: %s", marker)
	}

	tableStart := strings.LastIndex(text[:start], "{|")
	if tableStart == -1 {
		return "", fmt.Errorf("table start not found")
	}

	tableEndRel := strings.Index(text[tableStart:], "\n|}")
	if tableEndRel == -1 {
		return "", fmt.Errorf("table end not found")
	}

	tableEnd := tableStart + tableEndRel + len("\n|}")
	return text[tableStart:tableEnd], nil
}

func parseConstituentRows(table string) []Constituent {
	chunks := strings.Split(table, "\n|-")
	rows := make([]Constituent, 0)

	for _, chunk := range chunks {
		lines := strings.Split(chunk, "\n")
		cells := make([]string, 0)

		for _, line := range lines {
			line = strings.TrimSpace(line)

			if !strings.HasPrefix(line, "|") || strings.HasPrefix(line, "|}") {
				continue
			}

			line = strings.TrimPrefix(line, "|")
			parts := strings.Split(line, "||")
			for _, part := range parts {
				cells = append(cells, cleanWikiCell(part))
			}
		}

		if len(cells) < 4 {
			continue
		}

		symbol := cells[0]
		security := cells[1]

		if symbol == "" || symbol == "Symbol" || security == "" {
			continue
		}

		rows = append(rows, Constituent{
			Symbol:      symbol,
			Security:    security,
			Sector:      cells[2],
			SubIndustry: cells[3],
		})
	}

	return rows
}

func cleanWikiCell(s string) string {
	s = strings.TrimSpace(s)

	// {{NyseSymbol|MMM}} -> MMM
	templateRE := regexp.MustCompile(`\{\{[^|{}]+\|([^|{}]+)\}\}`)
	s = templateRE.ReplaceAllString(s, "$1")

	// [[3M]] -> 3M
	// [[Adobe Inc.|Adobe]] -> Adobe
	linkWithLabelRE := regexp.MustCompile(`\[\[[^|\]]+\|([^\]]+)\]\]`)
	s = linkWithLabelRE.ReplaceAllString(s, "$1")

	linkRE := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	s = linkRE.ReplaceAllString(s, "$1")

	s = strings.ReplaceAll(s, "'''", "")
	s = strings.ReplaceAll(s, "''", "")

	return strings.TrimSpace(s)
}

type Change struct {
	Date           string
	AddedSymbol    string
	AddedCompany   string
	RemovedSymbol  string
	RemovedCompany string
	Reason         string
}

func wikiChanges() {
	text := fetchWikiText()

	table, err := extractWikiTable(text, `id="changes"`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract table failed: %v\n", err)
		os.Exit(1)
	}

	rows := parseChangeRows(table)

	fmt.Printf("Fetched constituent changes: %d\n", len(rows))
	limit := 5
	if len(rows) < limit {
		limit = len(rows)
	}

	for i := 0; i < limit; i++ {
		fmt.Printf("\n%d.\n", i+1)
		fmt.Println(rows[i].Date)
		fmt.Printf("Added: %s %s\n", rows[i].AddedSymbol, rows[i].AddedCompany)
		fmt.Printf("Removed: %s %s\n", rows[i].RemovedSymbol, rows[i].RemovedCompany)
		fmt.Printf("Reason: %s\n", rows[i].Reason)
	}
}

func parseChangeRows(table string) []Change {
	chunks := strings.Split(table, "\n|-")
	rows := make([]Change, 0)

	for _, chunk := range chunks {
		lines := strings.Split(chunk, "\n")
		cells := make([]string, 0)

		for _, line := range lines {
			line = strings.TrimSpace(line)

			if !strings.HasPrefix(line, "|") || strings.HasPrefix(line, "|}") {
				continue
			}

			line = strings.TrimPrefix(line, "|")
			parts := strings.Split(line, "||")
			for _, part := range parts {
				cells = append(cells, cleanWikiCell(part))
			}
		}

		if len(cells) < 6 {
			continue
		}

		date := cells[0]
		if date == "" || strings.EqualFold(date, "Date") {
			continue
		}

		rows = append(rows, Change{
			Date:           date,
			AddedSymbol:    cells[1],
			AddedCompany:   cells[2],
			RemovedSymbol:  cells[3],
			RemovedCompany: cells[4],
			Reason:         cells[5],
		})
	}

	return rows
}

func wikiMembers() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: sp500ctl wiki members YYYY-MM-DD")
		os.Exit(2)
	}

	targetDate, err := time.Parse("2006-01-02", os.Args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid date %q, expected YYYY-MM-DD\n", os.Args[3])
		os.Exit(1)
	}

	text := fetchWikiText()

	constituentsTable, err := extractWikiTable(text, `id="constituents"`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract constituents table failed: %v\n", err)
		os.Exit(1)
	}

	changesTable, err := extractWikiTable(text, `id="changes"`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract changes table failed: %v\n", err)
		os.Exit(1)
	}

	current := parseConstituentRows(constituentsTable)
	changes := parseChangeRows(changesTable)

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

		// Replaying backward:
		// if X was added after targetDate, remove X
		// if Y was removed after targetDate, restore Y
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

	symbols := make([]string, 0, len(members))
	for symbol := range members {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	fmt.Printf("S&P 500 members on %s\n", targetDate.Format("2006-01-02"))
	fmt.Printf("count: %d\n\n", len(symbols))

	for _, symbol := range symbols {
		c := members[symbol]
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
