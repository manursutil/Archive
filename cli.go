package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// run dispatches a command to its handler.
func run(db *sql.DB, cmd string, args []string) {
	switch cmd {
	case "add":
		checkArgsLength(args)
		cmdAdd(db, args[0])
	case "list":
		cmdList(db)
	case "del":
		checkArgsLength(args)
		cmdDel(db, args[0])
	case "search":
		checkArgsLength(args)
		cmdSearch(db, args[0])
	case "serve":
		cmdServe(db, 8080)
	case "watch":
		checkArgsLength(args)
		cmdWatch(db, args[0])
	default:
		printUsage()
		os.Exit(1)
	}
}

func checkArgsLength(args []string) {
	if len(args) < 1 {
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

	if err := add(db, item); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func cmdList(db *sql.DB) {
	items, err := list(db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	for _, item := range items {
		fmt.Printf("%d: %s - %s [%s]\n", item.ID, item.Title, item.Source, item.SavedAt)
	}
}

func cmdDel(db *sql.DB, arg string) {
	id, err := strconv.Atoi(arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := del(db, id); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Item %d deleted successfully!", id)
}

func cmdSearch(db *sql.DB, arg string) {
	matches, err := search(db, arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	for _, match := range matches {
		fmt.Printf(
			"[%d] %s - %s\n    %s\n\n",
			match.ID,
			match.Title,
			match.Source,
			match.Snippet,
		)
	}
}

func scan(db *sql.DB, dir string, seen map[string]time.Time) {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if tp, _ := parseArg(path); tp != "path" {
			return nil
		}

		info, err := d.Info()
		if err != nil || !info.ModTime().After(seen[path]) {
			return nil
		}

		seen[path] = info.ModTime()

		item, err := getItem(path)
		if err == nil {
			err = upsert(db, item)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		}

		return nil
	})
}

func cmdWatch(db *sql.DB, dir string) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	seen := map[string]time.Time{}
	for {
		scan(db, dir, seen)
		time.Sleep(2 * time.Second)
	}
}
