package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// ./archive [add | list | del | search] <args>
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	db, err := openDB(dbPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	run(db, os.Args[1], os.Args[2:])
}

func printUsage() {
	fmt.Println(`Usage:
  - archive add <url | path/to/file>
  - archive list
  - archive del <id>
  - archive search 'terms to search'
	- archive serve`,
	)
}

// same database regardless of cwd. ARCHIVE_DB overrides it (tests use this).
func dbPath() string {
	if p := os.Getenv("ARCHIVE_DB"); p != "" {
		return p
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	dir = filepath.Join(dir, "archive")
	if err := os.MkdirAll(dir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	return filepath.Join(dir, "archive.db")
}
