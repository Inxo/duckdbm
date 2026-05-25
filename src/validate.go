package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

func validateMigrations(args []string, dir string) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		fmt.Printf("Failed to open validation database: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	var target string
	if len(args) > 1 {
		target = args[1]
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("Failed to read migrations directory: %v\n", err)
		os.Exit(1)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	hasErrors := false
	fmt.Println("Validating migrations... " + dir)

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		if target != "" && !strings.Contains(file.Name(), target) {
			continue
		}

		sqlContent, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			fmt.Printf("  ✗ %s — failed to read: %v\n", file.Name(), err)
			hasErrors = true
			continue
		}

		processed, err := processMacros(string(sqlContent))
		if err != nil {
			fmt.Printf("  ✗ %s — macro error: %v\n", file.Name(), err)
			hasErrors = true
			continue
		}

		parts := strings.Split(processed, "-- ROLLBACK")
		migrateSQL := strings.TrimSpace(strings.TrimPrefix(parts[0], "-- MIGRATE"))

		if verr := validateSection(db, migrateSQL); verr != nil {
			fmt.Printf("  ✗ %s — %v\n", file.Name(), verr)
			hasErrors = true
			continue
		}

		if len(parts) > 1 {
			rollbackSQL := strings.TrimSpace(parts[1])
			if rollbackSQL != "" {
				if verr := validateSection(db, rollbackSQL); verr != nil {
					fmt.Printf("  ✗ %s (ROLLBACK) — %v\n", file.Name(), verr)
					hasErrors = true
					continue
				}
			}
		}

		fmt.Printf("  ✓ %s\n", file.Name())
	}

	if hasErrors {
		fmt.Println("\nValidation failed.")
		os.Exit(1)
	}
	fmt.Println("\nAll migrations are valid.")
}

func validateSection(db *sql.DB, section string) error {
	for _, stmt := range strings.Split(section, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if _, err := db.Exec("EXPLAIN " + stmt); err != nil {
			msg := err.Error()
			if strings.Contains(msg, "Parser Error") ||
				strings.Contains(msg, "syntax error") ||
				strings.Contains(msg, "unexpected token") {
				return fmt.Errorf("syntax error: %v", err)
			}
		}
	}
	return nil
}
