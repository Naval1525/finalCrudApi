package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func InitDatabase(databaseURL string) (*sql.DB, error) {
	log.Print(databaseURL)
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	// Optional: Test the connection
	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func RunMigrations(db *sql.DB) error {
	// Create users table
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(query); err != nil {
		return err
	}

	// Create an index on email for faster lookups
	indexQuery := `CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`
	if _, err := db.Exec(indexQuery); err != nil {
		return err
	}

	log.Println("Database migrations completed")
	return nil
}
