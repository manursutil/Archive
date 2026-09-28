package main

import (
	"database/sql"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLI(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "archive")
	args := []string{"build", "-o", binary}

	// Share coverage data with go test so os.Exit paths count too.
	coverDir := ""

	if f := flag.Lookup("test.gocoverdir"); f != nil {
		coverDir = f.Value.String()
	}

	if coverDir != "" {
		args = append(args, "-cover")
	}

	args = append(args, ".")

	if out, err := exec.Command("go", args...).CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	invoke := func(t *testing.T, dir string, code int, want string, args ...string) string {
		t.Helper()
		cmd := exec.Command(binary, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOCOVERDIR="+coverDir, "ARCHIVE_DB=archive.db")

		out, err := cmd.CombinedOutput()
		gotCode := 0

		if err != nil {
			exit, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatal(err)
			}
			gotCode = exit.ExitCode()
		}

		if gotCode != code || !strings.Contains(string(out), want) {
			t.Fatalf("%v: exit %d, output %q; want exit %d containing %q", args, gotCode, out, code, want)
		}

		return string(out)
	}

	t.Run("workflow", func(t *testing.T) {
		dir := t.TempDir()

		if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("hello searchable world"), 0600); err != nil {
			t.Fatal(err)
		}

		invoke(t, dir, 0, "", "add", "note.txt")
		invoke(t, dir, 0, "1: note - note.txt [", "list")
		invoke(t, dir, 0, "[searchable]", "search", "searchable")
		invoke(t, dir, 0, "Item 1 deleted successfully!", "del", "1")

		if out := invoke(t, dir, 0, "", "search", "searchable"); out != "" {
			t.Fatalf("deleted item returned: %q", out)
		}
	})

	for _, args := range [][]string{nil, {"unknown"}, {"add"}, {"del"}, {"search"}} {
		t.Run("usage/"+strings.Join(args, "_"), func(t *testing.T) { invoke(t, t.TempDir(), 1, "Usage:", args...) })
	}

	for _, tc := range []struct {
		name, schema, want string
		args               []string
	}{
		{name: "invalid item", args: []string{"add", "invalid"}, want: "invalid argument"},
		{name: "invalid ID", args: []string{"del", "nope"}, want: "invalid syntax"},
		{name: "invalid query", args: []string{"search", `"`}, want: "unterminated string"},
		{name: "insert failure", schema: `CREATE TRIGGER reject_insert BEFORE INSERT ON items BEGIN SELECT RAISE(FAIL, 'insert blocked'); END`, args: []string{"add", "note.txt"}, want: "insert blocked"},
		{name: "list failure", schema: `DROP TABLE items; CREATE TABLE items (id, title, source)`, args: []string{"list"}, want: "no such column: savedAt"},
		{name: "delete failure", schema: `CREATE TRIGGER reject_delete BEFORE DELETE ON items BEGIN SELECT RAISE(FAIL, 'delete blocked'); END; INSERT INTO items (source, content) VALUES ('source', 'hello')`, args: []string{"del", "1"}, want: "delete blocked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("hello"), 0600); err != nil {
				t.Fatal(err)
			}

			if tc.schema != "" {
				db, err := sql.Open("sqlite", filepath.Join(dir, "archive.db"))
				if err != nil {
					t.Fatal(err)
				}
				if err := initDB(db); err != nil {
					t.Fatal(err)
				}
				execSQL(t, db, tc.schema)
				db.Close()
			}

			invoke(t, dir, 1, tc.want, tc.args...)
		})
	}
	t.Run("open failure", func(t *testing.T) {
		dir := t.TempDir()

		if err := os.Mkdir(filepath.Join(dir, "archive.db"), 0700); err != nil {
			t.Fatal(err)
		}

		invoke(t, dir, 1, "unable to open database", "list")
	})
}

func TestScan(t *testing.T) {
	db := testDB(t)
	if err := initDB(db); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	seen := map[string]time.Time{}

	found := func(term string) bool {
		t.Helper()
		res, err := search(db, term)
		if err != nil {
			t.Fatal(err)
		}
		return len(res) > 0
	}

	if err := os.WriteFile(path, []byte("original walrus"), 0600); err != nil {
		t.Fatal(err)
	}
	scan(db, dir, seen)
	if !found("walrus") {
		t.Fatal("new file not indexed")
	}

	if err := os.WriteFile(path, []byte("edited narwhal"), 0600); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
	scan(db, dir, seen)
	if !found("narwhal") || found("walrus") {
		t.Fatal("edited file not re-indexed")
	}

	items, err := list(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
}
