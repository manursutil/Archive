package main

import (
	"database/sql"
	"fmt"
	"os"
)

// run dispatches a command to its handler.
func run(db *sql.DB, cmd string, args []string) {
	switch cmd {
	case "add":
		cmdAdd(db, args[0])
	case "list":
		// list(db)
	case "del":
		// del(db, id)
	case "search":
		// search(db, string)
	default:
		printUsage()
		os.Exit(1)
	}
}

func cmdAdd(db *sql.DB, arg string) {
	// 1. Get the url or path from command line argument
	// 2. if url, get the contents of the page
	// 3. if file, check supported file types (.txt, .md, .pdf, docx, .odf)
	// and then parse the file
	// 4. turn into item object
	// 5. add to db
	item, err := getItem(arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	add(db, item)
}
