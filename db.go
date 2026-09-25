package main

import "database/sql"

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
		id PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL,
		content TEXT,
		savedAt DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	return err
}

func add(db *sql.DB, item Item) error {
	return nil
}
