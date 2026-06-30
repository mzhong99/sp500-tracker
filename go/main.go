package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(2)
	}

	switch {
	case os.Args[1] == "db" && os.Args[2] == "ping":
		dbPing()

	case os.Args[1] == "db" && os.Args[2] == "init":
		if err := dbInit(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("initialized database")

	case os.Args[1] == "wiki" && os.Args[2] == "ping":
		wikiPing()

	case os.Args[1] == "wiki" && os.Args[2] == "dump":
		wikiDump()

	case os.Args[1] == "wiki" && os.Args[2] == "current":
		wikiCurrent()

	case os.Args[1] == "wiki" && os.Args[2] == "changes":
		wikiChanges()

	case os.Args[1] == "wiki" && os.Args[2] == "members":
		wikiMembers(os.Args[3:])

	case os.Args[1] == "wiki" && os.Args[2] == "ingest":
		if err := wikiIngest(); err != nil {
			log.Fatal(err)
		}

	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  sp500ctl db ping")
	fmt.Fprintln(os.Stderr, "  sp500ctl db init")
	fmt.Fprintln(os.Stderr, "  sp500ctl wiki ping")
	fmt.Fprintln(os.Stderr, "  sp500ctl wiki ingest")
	fmt.Fprintln(os.Stderr, "  sp500ctl wiki dump")
	fmt.Fprintln(os.Stderr, "  sp500ctl wiki current")
	fmt.Fprintln(os.Stderr, "  sp500ctl wiki changes")
	fmt.Fprintln(os.Stderr, "  sp500ctl wiki members YYYY-MM-DD")
}

func wikiMembers(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: sp500ctl wiki members YYYY-MM-DD")
		os.Exit(2)
	}

	targetDate, err := time.Parse("2006-01-02", args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid date %q, expected YYYY-MM-DD\n", args[0])
		os.Exit(1)
	}

	current, changes := loadWikiData()
	members := ReplayMembers(current, changes, targetDate)

	PrintMembers(members, targetDate)
}
