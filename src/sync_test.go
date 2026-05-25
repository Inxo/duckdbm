package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordSyncMigration_StoresDuration(t *testing.T) {
	prev := dbFile
	t.Cleanup(func() {
		dbFile = prev
		os.Remove("test_recsync.db")
	})
	dbFile = "test_recsync.db"

	initialize()

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	recordSyncMigration(db, "001_import.sql", 1234)

	var filename string
	var durationMs int64
	err = db.QueryRow(
		"SELECT filename, duration_ms FROM attached_db.sync WHERE filename = '001_import.sql'",
	).Scan(&filename, &durationMs)
	if err != nil {
		t.Fatalf("query sync record: %v", err)
	}
	if filename != "001_import.sql" {
		t.Errorf("filename: want '001_import.sql', got %q", filename)
	}
	if durationMs != 1234 {
		t.Errorf("duration_ms: want 1234, got %d", durationMs)
	}
}

func TestRecordSyncMigration_StoresTimestamp(t *testing.T) {
	prev := dbFile
	t.Cleanup(func() {
		dbFile = prev
		os.Remove("test_synctime.db")
	})
	dbFile = "test_synctime.db"

	initialize()

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	before := time.Now().UTC().Add(-time.Second)
	recordSyncMigration(db, "ts_test.sql", 0)
	after := time.Now().UTC().Add(time.Second)

	var count int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM attached_db.sync WHERE filename='ts_test.sql' AND applied_at BETWEEN ? AND ?",
		before, after,
	).Scan(&count)
	if err != nil {
		t.Fatalf("timestamp query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row within time range, got %d", count)
	}
}

func TestSyncMigration_FileNotFound(t *testing.T) {
	prev, prevDir := dbFile, migrationsDir
	t.Cleanup(func() {
		dbFile = prev
		migrationsDir = prevDir
		os.Remove("test_syncnf.db")
	})
	dbFile = "test_syncnf.db"
	migrationsDir = "migrations"

	initialize()
	// Must not panic; prints error message
	syncMigration("999_does_not_exist")
}

func TestSyncMigration_RecordsEntry(t *testing.T) {
	dir := t.TempDir()
	prev, prevDir := dbFile, migrationsDir
	t.Cleanup(func() {
		dbFile = prev
		migrationsDir = prevDir
		os.Remove("test_syncok.db")
	})
	dbFile = "test_syncok.db"
	migrationsDir = dir

	initialize()

	err := os.WriteFile(filepath.Join(dir, "001_sync_me.sql"), []byte(`-- MIGRATE
CREATE TABLE synced_table (id INTEGER);
-- ROLLBACK
DROP TABLE synced_table;
`), 0644)
	if err != nil {
		t.Fatalf("write sync file: %v", err)
	}

	syncMigration("001_sync_me")

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM attached_db.sync WHERE filename='001_sync_me'").Scan(&count)
	if err != nil {
		t.Fatalf("query sync table: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 sync record, got %d", count)
	}
}

func TestSyncMigration_ExecutesOnlyMigrateSection(t *testing.T) {
	dir := t.TempDir()
	prev, prevDir := dbFile, migrationsDir
	t.Cleanup(func() {
		dbFile = prev
		migrationsDir = prevDir
		os.Remove("test_syncmig.db")
	})
	dbFile = "test_syncmig.db"
	migrationsDir = dir

	initialize()

	// ROLLBACK section drops the table; if it were executed the table would not exist
	err := os.WriteFile(filepath.Join(dir, "001_sections.sql"), []byte(`-- MIGRATE
CREATE TABLE section_test (id INTEGER);
-- ROLLBACK
DROP TABLE section_test;
`), 0644)
	if err != nil {
		t.Fatalf("write file: %v", err)
	}

	syncMigration("001_sections")

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='section_test'").Scan(&name)
	if err != nil {
		t.Errorf("section_test table not found — ROLLBACK section may have been executed: %v", err)
	}
}

func TestStartSpinner_StartsAndStops(t *testing.T) {
	done := startSpinner("test_op")
	time.Sleep(250 * time.Millisecond)
	close(done)
	time.Sleep(100 * time.Millisecond) // let goroutine clean up
}
