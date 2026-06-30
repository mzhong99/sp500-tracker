package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const wikiURL = "https://en.wikipedia.org/w/api.php?action=parse&page=List_of_S%26P_500_companies&prop=wikitext&format=json"

type WikiPage struct {
	Title string
	Text  string
}

type Constituent struct {
	Symbol      string
	Security    string
	Sector      string
	SubIndustry string
}

type Change struct {
	Date           time.Time
	AddedSymbol    string
	AddedCompany   string
	RemovedSymbol  string
	RemovedCompany string
	Reason         string
}

func wikiPing() {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", wikiURL, nil)
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
	page := fetchWikiPage()

	fmt.Printf("title: %s\n\n", page.Title)
	fmt.Print(page.Text)
}

func wikiCurrent() {
	current, _ := loadWikiData()

	fmt.Printf("Fetched current constituents: %d\n", len(current))
	if len(current) > 0 {
		fmt.Println("First:")
		fmt.Println(current[0].Symbol)
		fmt.Println(current[0].Security)
		fmt.Println(current[0].Sector)
		fmt.Println(current[0].SubIndustry)
	}
}

func wikiChanges() {
	_, changes := loadWikiData()

	fmt.Printf("Fetched constituent changes: %d\n", len(changes))

	limit := 5
	if len(changes) < limit {
		limit = len(changes)
	}

	for i := 0; i < limit; i++ {
		fmt.Printf("\n%d.\n", i+1)
		fmt.Println(changes[i].Date)
		fmt.Printf("Added: %s %s\n", changes[i].AddedSymbol, changes[i].AddedCompany)
		fmt.Printf("Removed: %s %s\n", changes[i].RemovedSymbol, changes[i].RemovedCompany)
		fmt.Printf("Reason: %s\n", changes[i].Reason)
	}
}

func loadWikiData() ([]Constituent, []Change) {
	text := fetchWikiPage().Text

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

	return parseConstituentRows(constituentsTable), parseChangeRows(changesTable)
}

func fetchWikiPage() WikiPage {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", wikiURL, nil)
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

	return WikiPage{
		Title: parsed.Parse.Title,
		Text:  parsed.Parse.Wikitext.Text,
	}
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
		cells := parseWikiCells(chunk)
		if len(cells) < 4 {
			continue
		}

		if cells[0] == "" || cells[0] == "Symbol" || cells[1] == "" {
			continue
		}

		rows = append(rows, Constituent{
			Symbol:      cells[0],
			Security:    cells[1],
			Sector:      cells[2],
			SubIndustry: cells[3],
		})
	}

	return rows
}

func parseChangeRows(table string) []Change {
	chunks := strings.Split(table, "\n|-")
	rows := make([]Change, 0)

	for _, chunk := range chunks {
		cells := parseWikiCells(chunk)
		if len(cells) < 6 {
			continue
		}

		if cells[0] == "" || strings.EqualFold(cells[0], "Date") {
			continue
		}

		rows = append(rows, Change{
			AddedSymbol:    cells[1],
			AddedCompany:   cells[2],
			RemovedSymbol:  cells[3],
			RemovedCompany: cells[4],
			Reason:         cells[5],
		})

		t, err := time.Parse("January 2, 2006", cells[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid date format %q: %v\n", cells[0], err)
			os.Exit(1)
		}
		rows[len(rows)-1].Date = t
	}

	return rows
}

func parseWikiCells(chunk string) []string {
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

	return cells
}

func cleanWikiCell(s string) string {
	s = strings.TrimSpace(s)

	templateRE := regexp.MustCompile(`\{\{[^|{}]+\|([^|{}]+)\}\}`)
	s = templateRE.ReplaceAllString(s, "$1")

	linkWithLabelRE := regexp.MustCompile(`\[\[[^|\]]+\|([^\]]+)\]\]`)
	s = linkWithLabelRE.ReplaceAllString(s, "$1")

	linkRE := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	s = linkRE.ReplaceAllString(s, "$1")

	s = strings.ReplaceAll(s, "'''", "")
	s = strings.ReplaceAll(s, "''", "")

	return strings.TrimSpace(s)
}
