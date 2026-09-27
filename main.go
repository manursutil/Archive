package main

import (
	"fmt"
	"os"
)

// ./archive [add | list | del | search] <args>
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// initialize db...
	db, err := openDB()
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
