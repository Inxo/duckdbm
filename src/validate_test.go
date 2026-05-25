package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open in-memory duckdb: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestValidateSection_ValidSelect(t *testing.T) {
	db := openMemDB(t)
	if err := validateSection(db, "SELECT 1"); err != nil {
		t.Errorf("expected no error for valid SELECT, got: %v", err)
	}
}

func TestValidateSection_MultipleStatements(t *testing.T) {
	db := openMemDB(t)
	section := "SELECT 1;\nSELECT 2;\nSELECT 3"
	if err := validateSection(db, section); err != nil {
		t.Errorf("expected no error for multiple valid statements, got: %v", err)
	}
}

func TestValidateSection_SyntaxError(t *testing.T) {
	db := openMemDB(t)
	if err := validateSection(db, "SELEKT * FRMO nowhere !!!"); err == nil {
		t.Error("expected error for syntax error, got nil")
	}
}

func TestValidateSection_RuntimeErrorIgnored(t *testing.T) {
	db := openMemDB(t)
	// "table not found" is a runtime error — validation must not flag it
	if err := validateSection(db, "SELECT * FROM nonexistent_table_xyz"); err != nil {
		t.Errorf("expected runtime error to be ignored, got: %v", err)
	}
}

func TestValidateSection_EmptySection(t *testing.T) {
	db := openMemDB(t)
	for _, s := range []string{"", "   ", "\n\t"} {
		if err := validateSection(db, s); err != nil {
			t.Errorf("empty section %q: expected no error, got: %v", s, err)
		}
	}
}

func TestValidateSection_CommentOnly(t *testing.T) {
	db := openMemDB(t)
	if err := validateSection(db, "-- this is just a comment"); err != nil {
		t.Errorf("comment-only section: expected no error, got: %v", err)
	}
}

func TestValidateSection_MixedValidAndComments(t *testing.T) {
	db := openMemDB(t)
	section := "-- setup\nSELECT 1;\n-- done"
	if err := validateSection(db, section); err != nil {
		t.Errorf("mixed comments/SQL: expected no error, got: %v", err)
	}
}

func TestValidateMigrations_AllValid(t *testing.T) {
	dir := t.TempDir()
	prevDir := migrationsDir
	migrationsDir = dir
	t.Cleanup(func() { migrationsDir = prevDir })

	files := map[string]string{
		"001_create.sql": "-- MIGRATE\nSELECT 1;\n-- ROLLBACK\nSELECT 2;\n",
		"002_insert.sql": "-- MIGRATE\nSELECT 'hello';\n-- ROLLBACK\nSELECT 'bye';\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Happy path: should print success and not call os.Exit
	validateMigrations([]string{"validate"})
}

func TestValidateMigrations_FiltersByTarget(t *testing.T) {
	dir := t.TempDir()
	prevDir := migrationsDir
	migrationsDir = dir
	t.Cleanup(func() { migrationsDir = prevDir })

	// Write one valid and one that would be invalid if parsed,
	// but won't be checked because the target filter excludes it.
	if err := os.WriteFile(filepath.Join(dir, "001_users.sql"),
		[]byte("-- MIGRATE\nSELECT 1;\n-- ROLLBACK\nSELECT 2;\n"), 0644); err != nil {
		t.Fatalf("write 001: %v", err)
	}
	// This file has a syntax error but the filter "users" won't match it
	if err := os.WriteFile(filepath.Join(dir, "002_orders.sql"),
		[]byte("-- MIGRATE\nSELEKT BAD SYNTAX!!!;\n-- ROLLBACK\nSELECT 1;\n"), 0644); err != nil {
		t.Fatalf("write 002: %v", err)
	}

	// Targeting "users" — should pass without touching 002_orders.sql
	validateMigrations([]string{"validate", "users"})
}

func TestValidateMigrations_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	prevDir := migrationsDir
	migrationsDir = dir
	t.Cleanup(func() { migrationsDir = prevDir })

	// No files — should print success without panic
	validateMigrations([]string{"validate"})
}

func TestValidateMigrations_IgnoresNonSQLFiles(t *testing.T) {
	dir := t.TempDir()
	prevDir := migrationsDir
	migrationsDir = dir
	t.Cleanup(func() { migrationsDir = prevDir })

	if err := os.WriteFile(filepath.Join(dir, "README.md"),
		[]byte("# not SQL"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}

	validateMigrations([]string{"validate"})
}
