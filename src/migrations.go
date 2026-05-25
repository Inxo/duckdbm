package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func initialize() {
	db, err := connectDB()
	if err != nil {
		fmt.Printf("Database connection error: %v\n", err)
		return
	}
	defer func(db *sql.DB) { _ = db.Close() }(db)

	if _, err = db.Exec(migrationsTableSQL); err != nil {
		fmt.Printf("Error creating migration table: %v\n", err)
		return
	}
	if _, err = db.Exec(syncTableSQL); err != nil {
		fmt.Printf("Error creating sync table: %v\n", err)
		return
	}
	fmt.Println("The database has been initialized..")
}

func createMigration(name string) {
	if err := os.MkdirAll(migrationsDir, os.ModePerm); err != nil {
		fmt.Printf("Error creating migrations folder: %v\n", err)
		return
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		fmt.Printf("Error reading migrations folder: %v\n", err)
		return
	}

	filename := fmt.Sprintf("%03d_%s.sql", len(files)+1, name)
	filePath := filepath.Join(migrationsDir, filename)

	if err = os.WriteFile(filePath, []byte("-- MIGRATE\n\n-- ROLLBACK\n"), 0644); err != nil {
		fmt.Printf("Error creating migration file: %v\n", err)
		return
	}
	fmt.Printf("Migration created: %s\n", filePath)
}

func applyMigrations() {
	db, err := connectDB()
	initialize()
	if err != nil {
		fmt.Printf("Failed to connect to the database: %v\n", err)
		return
	}
	defer func(db *sql.DB) { _ = db.Close() }(db)

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='migrations'").Scan(&tableName)
	if err == sql.ErrNoRows {
		fmt.Println("Migrations table not initialized. Run 'init' first.")
		return
	} else if err != nil {
		fmt.Printf("Failed to check migrations table: %v\n", err)
		return
	}

	rows, err := db.Query("SELECT filename FROM attached_db.migrations")
	if err != nil {
		fmt.Printf("Failed to fetch applied migrations: %v\n", err)
		return
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var filename string
		_ = rows.Scan(&filename)
		applied[filename] = true
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		fmt.Printf("Failed to read migrations directory: %v\n", err)
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") || applied[file.Name()] {
			continue
		}

		sqlContent, err := os.ReadFile(filepath.Join(migrationsDir, file.Name()))
		if err != nil {
			fmt.Printf("Failed to read file %s: %v\n", file.Name(), err)
			continue
		}

		processed, err := processMacros(string(sqlContent))
		if err != nil {
			fmt.Printf("Failed to process macros in file %s: %v\n", file.Name(), err)
			return
		}

		migrationSQL := strings.TrimSpace(strings.Split(processed, "-- ROLLBACK")[0])

		start := time.Now()
		_, err = db.Exec(migrationSQL)
		durationMs := time.Since(start).Milliseconds()
		if err != nil {
			fmt.Printf("Failed to apply migration %s: %v\n", file.Name(), err)
			break
		}

		if _, err = db.Exec("INSERT INTO attached_db.migrations (filename, duration_ms) VALUES (?, ?)", file.Name(), durationMs); err != nil {
			fmt.Printf("Failed to log migration %s: %v\n", file.Name(), err)
			break
		}

		fmt.Printf("Migration applied: %s (%dms)\n", file.Name(), durationMs)
	}
}

func rollbackLast(n int) {
	db, err := connectDB()
	initialize()
	if err != nil {
		fmt.Printf("Failed to connect to the database: %v\n", err)
		return
	}
	defer func(db *sql.DB) { _ = db.Close() }(db)

	rows, err := db.Query("SELECT id, filename FROM attached_db.migrations ORDER BY id DESC LIMIT ?", n)
	if err != nil {
		fmt.Printf("Failed to fetch applied migrations: %v\n", err)
		return
	}
	defer rows.Close()

	type migration struct {
		ID       int
		Filename string
	}
	var migrations []migration
	for rows.Next() {
		var m migration
		if err = rows.Scan(&m.ID, &m.Filename); err != nil {
			fmt.Printf("Failed to read migration row: %v\n", err)
			continue
		}
		migrations = append(migrations, m)
	}

	if len(migrations) == 0 {
		fmt.Println("No migrations to roll back.")
		return
	}

	for _, m := range migrations {
		sqlContent, err := os.ReadFile(filepath.Join(migrationsDir, m.Filename))
		if err != nil {
			fmt.Printf("Failed to read migration file %s: %v\n", m.Filename, err)
			continue
		}

		parts := strings.Split(string(sqlContent), "-- ROLLBACK")
		if len(parts) < 2 {
			fmt.Printf("No rollback section found in migration %s\n", m.Filename)
			continue
		}

		rollbackSQL, err := processMacros(strings.TrimSpace(parts[1]))
		if err != nil {
			fmt.Printf("Failed to process macros in rollback section of file %s: %v\n", m.Filename, err)
			continue
		}

		if _, err = db.Exec(rollbackSQL); err != nil {
			fmt.Printf("Failed to rollback migration %s: %v\n", m.Filename, err)
			break
		}
		if _, err = db.Exec("DELETE FROM attached_db.migrations WHERE id = ?", m.ID); err != nil {
			fmt.Printf("Failed to remove migration log %s: %v\n", m.Filename, err)
			break
		}
		fmt.Printf("Rolled back migration: %s\n", m.Filename)
	}
}

func listAppliedMigrations(args []string) {
	table := "migrations"
	limit := 10

	if len(args) > 1 {
		table = args[1]
	}
	if len(args) > 2 {
		n, err := strconv.Atoi(args[2])
		if err != nil {
			log.Fatalf("Invalid limit: %v", err)
		}
		limit = n
	}

	db, err := connectDB()
	if err != nil {
		fmt.Printf("Failed to connect to the database: %v\n", err)
		return
	}
	defer func(db *sql.DB) { _ = db.Close() }(db)

	var tableName string
	q := fmt.Sprintf("SELECT name FROM sqlite_master WHERE type='table' AND name='%s'", table)
	if err = db.QueryRow(q).Scan(&tableName); err == sql.ErrNoRows {
		fmt.Printf("'%s' table not initialized. Run 'init' first.\n", table)
		return
	} else if err != nil {
		fmt.Printf("Failed to check migrations table: %v\n", err)
		return
	}

	query := fmt.Sprintf("SELECT id, filename, applied_at, duration_ms FROM %s ORDER BY id DESC LIMIT %d", table, limit)
	rows, err := db.Query(query)
	if err != nil {
		fmt.Printf("Failed to fetch applied migrations: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Printf("Applied %s:\n", table)
	fmt.Println("ID\tFilename\t\tApplied At\t\tDuration")
	fmt.Println("----------------------------------------------------------------")
	for rows.Next() {
		var id int
		var filename string
		var appliedAt time.Time
		var durationMs sql.NullInt64
		if err = rows.Scan(&id, &filename, &appliedAt, &durationMs); err != nil {
			fmt.Printf("Failed to read migration row: %v\n", err)
			continue
		}
		durStr := "-"
		if durationMs.Valid {
			durStr = fmt.Sprintf("%dms", durationMs.Int64)
		}
		fmt.Printf("%d\t%s\t%s\t%s\n", id, filename, appliedAt.Format("2006-01-02 15:04:05"), durStr)
	}
}
