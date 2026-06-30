package main

import (
	"fmt"
	"log"
	"os"
	"strings"
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
	case os.Args[1] == "db" && os.Args[2] == "current":
		dbCurrent()

	case os.Args[1] == "db" && os.Args[2] == "changes":
		dbChanges()

	case os.Args[1] == "db" && os.Args[2] == "members":
		dbMembers(os.Args[3:])

	case os.Args[1] == "db" && os.Args[2] == "symbol":
		if len(os.Args) != 4 {
			fmt.Println("usage: sp500 db symbol SYMBOL")
			os.Exit(1)
		}

		symbol := strings.ToUpper(os.Args[3])

		if err := dbSymbol(symbol); err != nil {
			fmt.Println("db symbol failed:", err)
			os.Exit(1)
		}

	case os.Args[1] == "stooq":
		if err := handleStooqCommand(os.Args[2:]); err != nil {
			fmt.Println("stooq failed:", err)
			os.Exit(1)
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
	fmt.Fprintln(os.Stderr, "  sp500ctl db current")
	fmt.Fprintln(os.Stderr, "  sp500ctl db changes")
	fmt.Fprintln(os.Stderr, "  sp500ctl db members YYYY-MM-DD")
	fmt.Fprintln(os.Stderr, "  sp500ctl db symbol SYMBOL")
}
