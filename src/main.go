package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/marcboeker/go-duckdb" // Подключение DuckDB драйвера
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	migrationsTableSQL = `
CREATE SEQUENCE IF NOT EXISTS seq_id START 1;
CREATE TABLE IF NOT EXISTS migrations (
    id INTEGER PRIMARY KEY DEFAULT nextval('seq_id'),
    filename TEXT NOT NULL UNIQUE,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`
)

var migrationsDir = "migrations"

var dbFile string

func main() {
	// Processing command line arguments
	flag.StringVar(&dbFile, "db", "duckdb", "Database file (default 'duckdb')")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println("Usage: duckdbm [init|create|apply|rollback] [options]")
		return
	}

	command := flag.Args()[0]

	switch command {
	case "init":
		initialize()
	case "create":
		if len(flag.Args()) < 2 {
			fmt.Println("Input migration name.")
			return
		}
		createMigration(flag.Args()[1])
	case "apply":
		applyMigrations()
	case "rollback":
		rollbackLast()
	case "list":
		listAppliedMigrations()
	default:
		fmt.Printf("Unknown command: %s\n", command)
	}
}

func connectDB() (*sql.DB, error) {
	return sql.Open("duckdb", dbFile)
}

func initialize() {
	db, err := connectDB()
	if err != nil {
		fmt.Printf("Database connection error: %v\n", err)
		return
	}
	defer db.Close()

	_, err = db.Exec(migrationsTableSQL)
	if err != nil {
		fmt.Printf("Error creating migration table: %v\n", err)
		return
	}

	fmt.Println("The database has been initialized..")
}

func createMigration(name string) {
	err := os.MkdirAll(migrationsDir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating migrations folder: %v\n", err)
		return
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		fmt.Printf("Error reading migrations folder: %v\n", err)
		return
	}

	id := len(files) + 1
	filename := fmt.Sprintf("%03d_%s.sql", id, name)
	filepath := filepath.Join(migrationsDir, filename)

	err = os.WriteFile(filepath, []byte("-- MIGRATE\n\n-- ROLLBACK\n"), 0644)
	if err != nil {
		fmt.Printf("Error creating migration file: %v\n", err)
		return
	}

	fmt.Printf("Migration created: %s\n", filepath)
}

func applyMigrations() {
	db, err := connectDB()
	initialize()
	if err != nil {
		fmt.Printf("Failed to connect to the database: %v\n", err)
		return
	}
	defer db.Close()

	// Check if the migrations table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='migrations'").Scan(&tableName)
	if err == sql.ErrNoRows {
		fmt.Println("Migrations table not initialized. Run 'init' first.")
		return
	} else if err != nil {
		fmt.Printf("Failed to check migrations table: %v\n", err)
		return
	}

	// Fetch already applied migrations
	rows, err := db.Query("SELECT filename FROM migrations")
	if err != nil {
		fmt.Printf("Failed to fetch applied migrations: %v\n", err)
		return
	}
	defer rows.Close()

	appliedMigrations := make(map[string]bool)
	for rows.Next() {
		var filename string
		rows.Scan(&filename)
		appliedMigrations[filename] = true
	}

	// Read migration files from the directory
	files, err := ioutil.ReadDir(migrationsDir)
	if err != nil {
		fmt.Printf("Failed to read migrations directory: %v\n", err)
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// Apply migrations
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") || appliedMigrations[file.Name()] {
			continue
		}

		filepath := filepath.Join(migrationsDir, file.Name())
		sqlContent, err := os.ReadFile(filepath)
		if err != nil {
			fmt.Printf("Failed to read file %s: %v\n", file.Name(), err)
			continue
		}

		// Split migration SQL and rollback SQL
		parts := strings.Split(string(sqlContent), "-- ROLLBACK")
		migrationSQL := strings.TrimSpace(parts[0]) // Only apply the migration section

		_, err = db.Exec(migrationSQL)
		if err != nil {
			fmt.Printf("Failed to apply migration %s: %v\n", file.Name(), err)
			break
		}

		_, err = db.Exec("INSERT INTO migrations (filename) VALUES (?)", file.Name())
		if err != nil {
			fmt.Printf("Failed to log migration %s: %v\n", file.Name(), err)
			break
		}

		fmt.Printf("Migration applied: %s\n", file.Name())
	}
}

func rollbackLast() {
	db, err := connectDB()
	if err != nil {
		fmt.Printf("Database connection error: %v\n", err)
		return
	}
	defer db.Close()

	row := db.QueryRow("SELECT filename FROM migrations ORDER BY id DESC LIMIT 1")
	var filename string
	err = row.Scan(&filename)
	if err == sql.ErrNoRows {
		fmt.Println("No migrations to rollback.")
		return
	} else if err != nil {
		fmt.Printf("Error getting last migrationи: %v\n", err)
		return
	}

	filepath := filepath.Join(migrationsDir, filename)
	sqlContent, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("Error reading file %s: %v\n", filename, err)
		return
	}

	parts := strings.Split(string(sqlContent), "-- ROLLBACK")
	if len(parts) < 2 {
		fmt.Printf("Migration %s does not support rollback.\n", filename)
		return
	}

	rollbackSQL := strings.TrimSpace(parts[1])
	_, err = db.Exec(rollbackSQL)
	if err != nil {
		fmt.Printf("Migration rollback error %s: %v\n", filename, err)
		return
	}

	_, err = db.Exec("DELETE FROM migrations WHERE filename = ?", filename)
	if err != nil {
		fmt.Printf("Error deleting migration record: %v\n", err)
		return
	}

	fmt.Printf("Rollback migration: %s\n", filename)
}

func listAppliedMigrations() {
	db, err := connectDB()
	if err != nil {
		fmt.Printf("Failed to connect to the database: %v\n", err)
		return
	}
	defer db.Close()

	// Check if the migrations table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='migrations'").Scan(&tableName)
	if err == sql.ErrNoRows {
		fmt.Println("Migrations table not initialized. Run 'init' first.")
		return
	} else if err != nil {
		fmt.Printf("Failed to check migrations table: %v\n", err)
		return
	}

	// Query applied migrations
	rows, err := db.Query("SELECT id, filename, applied_at FROM migrations ORDER BY id")
	if err != nil {
		fmt.Printf("Failed to fetch applied migrations: %v\n", err)
		return
	}
	defer rows.Close()

	// Display applied migrations
	fmt.Println("Applied Migrations:")
	fmt.Println("ID\tFilename\t\tApplied At")
	fmt.Println("------------------------------------------------")
	for rows.Next() {
		var id int
		var filename string
		var appliedAt time.Time
		err = rows.Scan(&id, &filename, &appliedAt)
		if err != nil {
			fmt.Printf("Failed to read migration row: %v\n", err)
			continue
		}
		fmt.Printf("%d\t%s\t%s\n", id, filename, appliedAt.Format("2006-01-02 15:04:05"))
	}
}
