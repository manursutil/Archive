package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { db.Close() })
	return db
}

func execSQL(t *testing.T, db *sql.DB, query string) {
	t.Helper()

	if _, err := db.Exec(query); err != nil {
		t.Fatal(err)
	}
}

func TestDatabase(t *testing.T) {
	t.Chdir(t.TempDir())

	db, err := openDB("archive.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		t.Fatalf("schema not idempotent: %v", err)
	}

	items, err := list(db)
	if err != nil || len(items) != 0 {
		t.Fatalf("empty list = %v, %v", items, err)
	}

	for i := 0; i < 12; i++ {
		if err := add(db, Item{Title: fmt.Sprintf("note %d", i), Source: fmt.Sprintf("note-%d.txt", i), Content: "hello searchable world"}); err != nil {
			t.Fatal(err)
		}
	}

	if err := add(db, Item{Source: "https://example.com", Content: "page", HTML: "<p>page</p>"}); err != nil {
		t.Fatal(err)
	}

	if page, err := getHTML(db, 13); err != nil || page != "<p>page</p>" {
		t.Fatalf("getHTML = %q, %v", page, err)
	}

	if _, err := getHTML(db, 1); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("file has HTML: %v", err)
	}

	if text, err := getText(db, 1); err != nil || text != "hello searchable world" {
		t.Fatalf("getText = %q, %v", text, err)
	}

	if _, err := getText(db, 999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing item has text: %v", err)
	}

	items, err = list(db)
	if err != nil || len(items) != 10 {
		t.Fatalf("list limit = %v, %v", items, err)
	}

	if items[0].ID != 1 || items[0].Title != "note 0" || items[0].Source != "note-0.txt" || items[0].SavedAt == "" || items[0].Excerpt != "hello searchable world" || items[0].HasHTML {
		t.Fatalf("bad item: %+v", items[0])
	}

	matches, err := search(db, "searchable")
	if err != nil || len(matches) != 12 {
		t.Fatalf("search = %v, %v", matches, err)
	}

	if matches[0].ID != 1 || matches[0].Title != "note 0" || matches[0].Source != "note-0.txt" || !strings.Contains(matches[0].Snippet, "[searchable]") || matches[0].Score >= 0 || matches[0].HasHTML {
		t.Fatalf("bad match: %+v", matches[0])
	}

	if err := del(db, 1); err != nil {
		t.Fatal(err)
	}

	if err := del(db, 999); err != nil {
		t.Fatal(err)
	}

	matches, err = search(db, "searchable")
	if err != nil || len(matches) != 11 {
		t.Fatalf("delete did not update index: %v, %v", matches, err)
	}

	for _, match := range matches {
		if match.ID == 1 {
			t.Fatal("deleted item still indexed")
		}
	}

	matches, err = search(db, "absent")
	if err != nil || len(matches) != 0 {
		t.Fatalf("no matches = %v, %v", matches, err)
	}

	if _, err := search(db, `"`); err == nil {
		t.Fatal("invalid search accepted")
	}
}

func TestDatabaseErrors(t *testing.T) {
	t.Run("closed", func(t *testing.T) {
		db := testDB(t)
		db.Close()

		if err := initDB(db); err == nil {
			t.Fatal("init succeeded")
		}

		if err := add(db, Item{}); err == nil {
			t.Fatal("add succeeded")
		}

		if _, err := list(db); err == nil {
			t.Fatal("list succeeded")
		}

		if err := del(db, 1); err == nil || !strings.Contains(err.Error(), "delete error:") {
			t.Fatalf("del = %v", err)
		}

		if _, err := search(db, "hello"); err == nil {
			t.Fatal("search succeeded")
		}
	})
	t.Run("list scan", func(t *testing.T) {
		db := testDB(t)

		execSQL(t, db, `CREATE TABLE items (id, title, source, savedAt, content, html); INSERT INTO items VALUES ('bad', 'title', 'source', 'today', '', '')`)

		if _, err := list(db); err == nil {
			t.Fatal("invalid ID accepted")
		}
	})
	t.Run("list iteration", func(t *testing.T) {
		db := testDB(t)

		execSQL(t, db, `CREATE VIEW items AS SELECT 1 AS id, 'title' AS title, 'source' AS source, 'today' AS savedAt, '' AS content, '' AS html UNION ALL SELECT abs(-9223372036854775808), 'title', 'source', 'today', '', ''`)

		if _, err := list(db); err == nil || !strings.Contains(err.Error(), "integer overflow") {
			t.Fatalf("list = %v", err)
		}
	})
	t.Run("search scan", func(t *testing.T) {
		db := testDB(t)

		execSQL(t, db, `CREATE TABLE items (id INTEGER PRIMARY KEY, title, source, content, html, savedAt)`)

		if err := initDB(db); err != nil {
			t.Fatal(err)
		}

		execSQL(t, db, `INSERT INTO items (source, content) VALUES (NULL, 'hello')`)

		if _, err := search(db, "hello"); err == nil {
			t.Fatal("NULL source accepted")
		}
	})
	t.Run("fts schema", func(t *testing.T) {
		db := testDB(t)
		execSQL(t, db, `CREATE TABLE items (id INTEGER PRIMARY KEY, source, content); CREATE INDEX items_fts ON items(content)`)

		if err := initDB(db); err == nil {
			t.Fatal("conflicting schema accepted")
		}
	})

	for _, mode := range []string{"directory", "schema"} {
		t.Run("open "+mode, func(t *testing.T) {
			t.Chdir(t.TempDir())

			if mode == "directory" {
				if err := os.Mkdir("archive.db", 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				db, err := sql.Open("sqlite", "archive.db")
				if err != nil {
					t.Fatal(err)
				}

				execSQL(t, db, `CREATE VIEW items AS SELECT 1`)
				db.Close()
			}

			db, err := openDB("archive.db")
			if db != nil {
				db.Close()
			}

			if err == nil {
				t.Fatal("invalid database opened")
			}
		})
	}
}
