package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	migrationsTableSQL = `
CREATE SEQUENCE IF NOT EXISTS attached_db.seq_id START 1;
CREATE TABLE IF NOT EXISTS attached_db.migrations (
    id INTEGER PRIMARY KEY DEFAULT nextval('attached_db.seq_id'),
    filename TEXT NOT NULL UNIQUE,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER
);
`
	syncTableSQL = `
CREATE SEQUENCE IF NOT EXISTS attached_db.seq_sync_id START 1;
CREATE TABLE IF NOT EXISTS attached_db.sync (
    id INTEGER PRIMARY KEY DEFAULT nextval('attached_db.seq_sync_id'),
    filename TEXT NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER
);
`
)

var migrationsDir = "migrations"
var dbFile string

func connectDB() (*sql.DB, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, err
	}

	encKey := ""
	if key := os.Getenv("ENC_KEY"); key != "" {
		encKey = fmt.Sprintf("(ENCRYPTION_KEY '%s')", key)
	}

	attachQuery := fmt.Sprintf(
		"USE memory; DETACH DATABASE IF EXISTS attached_db; ATTACH IF NOT EXISTS DATABASE '%s' AS attached_db %s; USE attached_db;",
		dbFile, encKey,
	)
	if _, err = db.Exec(attachQuery); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to attach database: %v", err)
	}
	return db, nil
}

func isSyncTableInitialized() bool {
	db, err := connectDB()
	if err != nil {
		fmt.Printf("Failed to connect to the database: %v\n", err)
		return false
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Printf("Failed to close the database: %v\n", err)
		}
	}()
	_, err = db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name='sync'`)
	return err == nil
}
