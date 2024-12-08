package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
)

const testMigrationsDir = "test_migrations"
const testDBFile = "test.db"

func setupTestDatabase(t *testing.T, i bool) *sql.DB {
	dbFile = testDBFile
	db, err := sql.Open("duckdb", dbFile)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if i != false {
		_, err = db.Exec(migrationsTableSQL)
		if err != nil {
			t.Fatalf("Failed to create migrations table: %v", err)
		}
	}

	return db
}

func setupTestMigrationsDir(t *testing.T) {
	err := os.Mkdir(testMigrationsDir, 0755)
	if err != nil && !os.IsExist(err) {
		t.Fatalf("Failed to create test migrations directory: %v", err)
	}
	migrationsDir = testMigrationsDir
}

func teardownTestMigrationsDir(t *testing.T) {
	err := os.RemoveAll(testMigrationsDir)
	if err != nil {
		t.Fatalf("Failed to clean up test migrations directory: %v", err)
	}
}

func TestCreateMigration(t *testing.T) {
	setupTestMigrationsDir(t)
	defer teardownTestMigrationsDir(t)

	createMigration("add_test_table")
	files, err := os.ReadDir(testMigrationsDir)
	if err != nil {
		t.Fatalf("Failed to read test migrations directory: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 migration file, got %d", len(files))
	}

	expectedFileName := "001_add_test_table.sql"
	if files[0].Name() != expectedFileName {
		t.Fatalf("Expected migration file %s, got %s", expectedFileName, files[0].Name())
	}
}

func teardownTestDb() {
	os.Remove(testDBFile)
}

func TestApplyMigrations(t *testing.T) {
	defer teardownTestDb()
	initialize()
	db := setupTestDatabase(t, true)
	defer db.Close()

	setupTestMigrationsDir(t)
	defer teardownTestMigrationsDir(t)

	// Create a sample migration
	migrationFile := filepath.Join(testMigrationsDir, "001_create_test_table.sql")
	err := os.WriteFile(migrationFile, []byte(`
		-- MIGRATE
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		-- ROLLBACK
		DROP TABLE test_table;
	`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test migration file: %v", err)
	}

	applyMigrations()
	db = setupTestDatabase(t, false)

	// Verify that the migration was applied
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Expected test_table to be created: %v", err)
	}

	// Verify that the migration was logged
	var filename string
	err = db.QueryRow("SELECT filename FROM migrations WHERE filename='001_create_test_table.sql'").Scan(&filename)
	if err != nil {
		t.Fatalf("Migration was not logged: %v", err)
	}
}

func TestListAppliedMigrations(t *testing.T) {
	db := setupTestDatabase(t, true)
	defer db.Close()
	defer teardownTestDb()
	initialize()

	_, err := db.Exec("INSERT INTO migrations (filename) VALUES ('001_test_migration.sql')")
	if err != nil {
		t.Fatalf("Failed to insert test migration: %v", err)
	}

	listAppliedMigrations() // Should display the applied migration in stdout
}
