package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func startSpinner(name string) chan struct{} {
	done := make(chan struct{})
	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		start := time.Now()
		i := 0
		for {
			select {
			case <-done:
				fmt.Print("\r\033[K")
				return
			default:
				fmt.Printf("\r%s Syncing %s... (%.1fs)", frames[i%len(frames)], name, time.Since(start).Seconds())
				i++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
	return done
}

func syncMigration(migrationName string) {
	if !isSyncTableInitialized() {
		fmt.Println("Error: Migrations table is not initialized. Run 'init' first.")
		return
	}

	migrationFile := filepath.Join("migrations", fmt.Sprintf("%s.sql", migrationName))
	if _, err := os.Stat(migrationFile); os.IsNotExist(err) {
		fmt.Printf("Error: Migration file %s not found.\n", migrationFile)
		return
	}

	sqlContent, err := os.ReadFile(migrationFile)
	if err != nil {
		fmt.Printf("Error reading migration file %s: %v\n", migrationFile, err)
		return
	}

	processed, err := processMacros(string(sqlContent))
	if err != nil {
		fmt.Printf("Failed to process macros in file %s: %v\n", migrationFile, err)
		return
	}
	sqlStatements := strings.Split(processed, "-- ROLLBACK")[0]

	db, err := connectDB()
	if err != nil {
		fmt.Printf("Failed to connect to the database: %v\n", err)
		return
	}
	defer func(db *sql.DB) { _ = db.Close() }(db)

	done := startSpinner(migrationName)
	start := time.Now()
	_, err = db.Exec(sqlStatements)
	durationMs := time.Since(start).Milliseconds()
	close(done)
	time.Sleep(50 * time.Millisecond)

	if err != nil {
		fmt.Printf("✗ Error syncing %s: %v\n", migrationName, err)
		return
	}

	recordSyncMigration(db, migrationName, durationMs)
	fmt.Printf("✓ Successfully synced: %s (%.3fs)\n", migrationName, float64(durationMs)/1000)
}

func recordSyncMigration(db *sql.DB, migrationName string, durationMs int64) {
	_, err := db.Exec(
		`INSERT INTO attached_db.sync (filename, applied_at, duration_ms) VALUES (?, ?, ?)`,
		migrationName, time.Now().UTC(), durationMs,
	)
	if err != nil {
		fmt.Printf("Error recording synced migration: %v\n", err)
	}
}
