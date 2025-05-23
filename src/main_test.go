package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/marcboeker/go-duckdb/v2"
)

const testMigrationsDir = "test_migrations"
const testDBFile = "test.db"

func setupTestDatabase(t *testing.T, i bool) *sql.DB {
	dbFile = testDBFile
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	// Подключаем базу данных через ATTACH с именем "attached_db"
	attachQuery := fmt.Sprintf("Use main; DETACH DATABASE IF EXISTS attached_db; ATTACH DATABASE '%s' AS attached_db; USE attached_db;", dbFile)
	_, err = db.Exec(attachQuery)
	if err != nil {
		//attachQuery := fmt.Sprintf("USE main; DETACH attached_db;")
		//_, err = db.Exec(attachQuery)
		_ = db.Close()
		t.Fatalf("Cannot attach: %v", err)
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

func teardownTestDb(db *sql.DB) {
	_ = db.Close()
	_ = os.Remove(testDBFile)
}

func TestApplyMigrations(t *testing.T) {
	//initialize()
	db := setupTestDatabase(t, true)
	defer teardownTestDb(db)
	defer func(db *sql.DB) {
		_ = db.Close()
	}(db)

	setupTestMigrationsDir(t)
	defer teardownTestMigrationsDir(t)

	// Create a sample migration
	migrationFile := filepath.Join(testMigrationsDir, "001_create_test_table.sql")
	err := os.WriteFile(migrationFile, []byte(`
		-- MIGRATE
		CREATE TABLE attached_db.test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);
		-- ROLLBACK
		DROP TABLE attached_db.test_table;
	`), 0644)
	if err != nil {
		t.Fatalf("Failed to write test migration file: %v", err)
	}

	//teardownTestDb()
	_ = db.Close()
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
	err = db.QueryRow("SELECT filename FROM attached_db.migrations WHERE filename='001_create_test_table.sql'").Scan(&filename)
	if err != nil {
		t.Fatalf("Migration was not logged: %v", err)
	}

	_ = db.Close()
	// Verify mysql load
	//_, err = db.Query("install mysql; load mysql;")
	//if err != nil {
	//	t.Fatalf("Mysql not loaded: %v", err)
	//}
}

func TestListAppliedMigrations(t *testing.T) {
	db := setupTestDatabase(t, true)
	defer func(db *sql.DB) {
		_ = db.Close()
	}(db)
	defer teardownTestDb(db)
	initialize()

	_, err := db.Exec("INSERT INTO attached_db.migrations (filename) VALUES ('001_test_migration.sql')")
	if err != nil {
		t.Fatalf("Failed to insert test migration: %v", err)
	}

	var agrs []string
	listAppliedMigrations(agrs) // Should display the applied migration in stdout
}

func TestRollbackLast(t *testing.T) {
	setupTestMigrationsDir(t)
	defer teardownTestMigrationsDir(t)

	// Prepare sample migrations
	migration1 := `
	CREATE TABLE attached_db.test_table1 (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL
	);
	-- ROLLBACK
	DROP TABLE attached_db.test_table1;
	`
	migration2 := `
	CREATE TABLE attached_db.test_table2 (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL
	);
	-- ROLLBACK
	DROP TABLE attached_db.test_table2;
	`

	// Write migration files
	err := os.WriteFile(filepath.Join(testMigrationsDir, "001_test_migration1.sql"), []byte(migration1), 0644)
	if err != nil {
		t.Fatalf("Failed to write test migration1: %v", err)
	}
	err = os.WriteFile(filepath.Join(testMigrationsDir, "002_test_migration2.sql"), []byte(migration2), 0644)
	if err != nil {
		t.Fatalf("Failed to write test migration2: %v", err)
	}

	// Apply migrations
	db := setupTestDatabase(t, true)

	defer teardownTestDb(db)
	err = db.Close()
	if err != nil {
		t.Fatalf("Failed to close db: %v", err)
	}
	applyMigrations()
	db = setupTestDatabase(t, false)

	// Ensure tables exist
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table1'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Expected test_table1 to exist: %v", err)
	}
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table2'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Expected test_table2 to exist: %v", err)
	}
	err = db.Close()
	if err != nil {
		t.Fatalf("Failed to close db: %v", err)
	}

	// Test rollback of the last migration
	rollbackLast(1)

	db = setupTestDatabase(t, false)

	// Ensure the second table is dropped and the first remains
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table2'").Scan(&tableName)
	if err == nil {
		t.Fatalf("Expected test_table2 to be dropped, but it still exists.")
	}
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table1'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Expected test_table1 to exist: %v", err)
	}

	_ = db.Close()
	// Rollback the remaining migration
	rollbackLast(1)
	db = setupTestDatabase(t, false)

	// Ensure both tables are dropped
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table1'").Scan(&tableName)
	if err == nil {
		t.Fatalf("Expected test_table1 to be dropped, but it still exists.")
	}

	// Verify no more migrations exist
	row := db.QueryRow("SELECT COUNT(*) FROM attached_db.migrations")
	var count int
	err = row.Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count remaining migrations: %v", err)
	}
	if count != 0 {
		t.Fatalf("Expected 0 remaining migrations, found %d", count)
	}
}

func TestRollbackLastInvalidCount(t *testing.T) {
	initialize()
	db := setupTestDatabase(t, true)
	defer teardownTestDb(db)
	defer func(db *sql.DB) {
		_ = db.Close()
	}(db)

	// Test rollback with invalid count
	defer func() {
		if r := recover(); r == nil {
			return
		}
	}()

	rollbackLast(-1)
}
