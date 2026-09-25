package main

import (
	"database/sql"
)

func openDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "archive.db")
	if err != nil {
		return nil, err
	}

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
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS items (
		id INT PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL,
		content TEXT,
		savedAt DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	return err
}

func add(db *sql.DB, item Item) error {
	_, err := db.Exec(`INSERT INTO items (url, content) VALUES (?, ?)`, item.Source, item.Content)

	return err
}

func list(db *sql.DB) ([]ItemRepr, error) {
	query := `SELECT id, source, savedAt FROM items LIMIT 10`

	rows, err := db.Query(query)
	if err != nil {
		return []ItemRepr{}, err
	}
	defer rows.Close()

	items := make([]ItemRepr, 0)

	for rows.Next() {
		var id int
		var source string
		var savedAt string

		err := rows.Scan(&id, &source, &savedAt)
		if err != nil {
			return []ItemRepr{}, err
		}

		items = append(items, ItemRepr{
			ID:      id,
			Source:  source,
			SavedAt: savedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return []ItemRepr{}, err
	}

	return items, nil
}
