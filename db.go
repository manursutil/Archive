package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func openDB(path string) (*sql.DB, error) {
	// The registered SQLite driver opens the connection on Ping, not sql.Open.
	db, _ := sql.Open("sqlite", path)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	if err := initDB(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func initDB(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NULL,
		source TEXT UNIQUE NOT NULL,
		content TEXT,
		html TEXT,
		savedAt DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(
		content,
		content='items',
		content_rowid='id'
	)`); err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE TRIGGER IF NOT EXISTS items_ai AFTER INSERT ON items BEGIN
		INSERT INTO items_fts(rowid, content) VALUES (new.id, new.content);
	END;
	CREATE TRIGGER IF NOT EXISTS items_ad AFTER DELETE ON items BEGIN
		INSERT INTO items_fts(items_fts, rowid, content) VALUES ('delete', old.id, old.content);
	END;`); err != nil {
		return err
	}

	_, err := db.Exec(`CREATE TRIGGER IF NOT EXISTS items_au AFTER UPDATE ON items BEGIN
	INSERT INTO items_fts(items_fts, rowid, content) VALUES ('delete', old.id, old.content);
	INSERT INTO items_fts(rowid, content) VALUES (new.id, new.content);
	END;`)

	return err
}

func add(db *sql.DB, item Item) error {
	_, err := db.Exec(`INSERT INTO items (title, source, content, html) VALUES (?, ?, ?, ?)`, item.Title, item.Source, item.Content, item.HTML)

	return err
}

func list(db *sql.DB) ([]ItemRepr, error) {
	query := `SELECT id, COALESCE(title, ''), source, savedAt, substr(COALESCE(content, ''), 1, 300), COALESCE(html, '') != '' FROM items LIMIT 10`

	rows, err := db.Query(query)
	if err != nil {
		return []ItemRepr{}, err
	}
	defer rows.Close()

	items := make([]ItemRepr, 0)

	for rows.Next() {
		var item ItemRepr

		err := rows.Scan(&item.ID, &item.Title, &item.Source, &item.SavedAt, &item.Excerpt, &item.HasHTML)
		if err != nil {
			return []ItemRepr{}, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return []ItemRepr{}, err
	}

	return items, nil
}

func getHTML(db *sql.DB, id int) (string, error) {
	var page string
	err := db.QueryRow(`SELECT html FROM items WHERE id = ? AND html != ''`, id).Scan(&page)

	return page, err
}

func getText(db *sql.DB, id int) (string, error) {
	var text string
	err := db.QueryRow(`SELECT COALESCE(content, '') FROM items WHERE id = ?`, id).Scan(&text)

	return text, err
}

func del(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete error: %v", err)
	}

	return nil
}

func search(db *sql.DB, searchTerm string) ([]SearchResult, error) {
	rows, err := db.Query(`
		SELECT
			i.id,
			COALESCE(i.title, ''),
			i.source,
			snippet(items_fts, 0, '[', ']', '...', 20),
			bm25(items_fts),
			COALESCE(i.html, '') != ''
		FROM items_fts
		JOIN items i ON i.id = items_fts.rowid
		WHERE items_fts MATCH ?
		ORDER BY bm25(items_fts)
	`, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]SearchResult, 0)

	for rows.Next() {
		var result SearchResult

		err := rows.Scan(&result.ID, &result.Title, &result.Source, &result.Snippet, &result.Score, &result.HasHTML)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

func upsert(db *sql.DB, item Item) error {
	_, err := db.Exec(`INSERT INTO items (title, source, content, html) VALUES (?, ?, ?, ?)
	ON CONFLICT(source) DO UPDATE SET title=excluded.title, content=excluded.content, html=excluded.html, savedAt=CURRENT_TIMESTAMP`, item.Title, item.Source, item.Content, item.HTML)
	return err
}
