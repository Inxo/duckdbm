package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetGlobals saves and restores the two mutable globals used by migrations.
func resetGlobals(t *testing.T, db, dir string) {
	t.Helper()
	prevDB, prevDir := dbFile, migrationsDir
	dbFile = db
	migrationsDir = dir
	t.Cleanup(func() {
		dbFile = prevDB
		migrationsDir = prevDir
		os.Remove(db)
		os.RemoveAll(dir)
	})
}

func TestInitialize_CreatesBothTables(t *testing.T) {
	dir := t.TempDir()
	resetGlobals(t, "test_init.db", dir)

	initialize()

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB after initialize: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"migrations", "sync"} {
		var name string
		err = db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found after initialize: %v", table, err)
		}
	}
}

func TestApplyMigrations_RecordsDurationMs(t *testing.T) {
	dir := t.TempDir()
	resetGlobals(t, "test_duration.db", dir)

	err := os.WriteFile(filepath.Join(dir, "001_dur.sql"), []byte(`-- MIGRATE
CREATE TABLE dur_table (id INTEGER);
-- ROLLBACK
DROP TABLE dur_table;
`), 0644)
	if err != nil {
		t.Fatalf("write migration: %v", err)
	}

	initialize()
	applyMigrations()

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	var durMs int64
	err = db.QueryRow(
		"SELECT duration_ms FROM attached_db.migrations WHERE filename = '001_dur.sql'",
	).Scan(&durMs)
	if err != nil {
		t.Fatalf("query duration_ms: %v", err)
	}
	if durMs < 0 {
		t.Errorf("expected duration_ms >= 0, got %d", durMs)
	}
}

func TestApplyMigrations_SkipsAlreadyApplied(t *testing.T) {
	dir := t.TempDir()
	resetGlobals(t, "test_skip.db", dir)

	err := os.WriteFile(filepath.Join(dir, "001_skip.sql"), []byte(`-- MIGRATE
CREATE TABLE skip_table (id INTEGER);
-- ROLLBACK
DROP TABLE skip_table;
`), 0644)
	if err != nil {
		t.Fatalf("write migration: %v", err)
	}

	initialize()
	applyMigrations()
	applyMigrations() // second call — must not fail or re-apply

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	var count int
	if err = db.QueryRow("SELECT COUNT(*) FROM attached_db.migrations WHERE filename='001_skip.sql'").Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected migration logged exactly once, got %d", count)
	}
}

func TestApplyMigrations_SubstitutesMacros(t *testing.T) {
	dir := t.TempDir()
	resetGlobals(t, "test_macro.db", dir)
	t.Setenv("TEST_TABLE_NAME", "macro_table")

	err := os.WriteFile(filepath.Join(dir, "001_macro.sql"), []byte(`-- MIGRATE
CREATE TABLE {{TEST_TABLE_NAME}} (id INTEGER);
-- ROLLBACK
DROP TABLE {{TEST_TABLE_NAME}};
`), 0644)
	if err != nil {
		t.Fatalf("write migration: %v", err)
	}

	initialize()
	applyMigrations()

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='macro_table'").Scan(&name)
	if err != nil {
		t.Errorf("macro_table not created: %v", err)
	}
}

func TestRollbackLast_EmptyDB(t *testing.T) {
	dir := t.TempDir()
	resetGlobals(t, "test_rb_empty.db", dir)

	initialize()
	// Should print "No migrations to roll back." without panicking
	rollbackLast(1)
}

func TestRollbackLast_MissingRollbackSection(t *testing.T) {
	dir := t.TempDir()
	resetGlobals(t, "test_rb_nosection.db", dir)

	// File has no -- ROLLBACK marker
	err := os.WriteFile(filepath.Join(dir, "001_norollback.sql"), []byte(`-- MIGRATE
CREATE TABLE no_rb (id INTEGER);
`), 0644)
	if err != nil {
		t.Fatalf("write migration: %v", err)
	}

	initialize()
	applyMigrations()
	// Should print a warning but not panic
	rollbackLast(1)
}

func TestListAppliedMigrations_ShowsDuration(t *testing.T) {
	dir := t.TempDir()
	resetGlobals(t, "test_list_dur.db", dir)

	initialize()

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO attached_db.migrations (filename, duration_ms) VALUES ('001_test.sql', 42)")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	db.Close()

	// Smoke test: no panic, no crash
	listAppliedMigrations([]string{})
}

func TestListAppliedMigrations_SyncTable(t *testing.T) {
	dir := t.TempDir()
	resetGlobals(t, "test_list_sync.db", dir)

	initialize()

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO attached_db.sync (filename, duration_ms) VALUES ('001_sync.sql', 99)")
	if err != nil {
		t.Fatalf("insert sync: %v", err)
	}
	db.Close()

	listAppliedMigrations([]string{"list", "sync"})
}

func TestListAppliedMigrations_RespectsLimit(t *testing.T) {
	dir := t.TempDir()
	resetGlobals(t, "test_list_limit.db", dir)

	initialize()

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB: %v", err)
	}
	defer db.Close()

	for i := 1; i <= 5; i++ {
		_, err = db.Exec("INSERT INTO attached_db.migrations (filename) VALUES (?)",
			strings.Repeat("a", i)+"_migration.sql")
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	db.Close()

	// limit=2, should not panic
	listAppliedMigrations([]string{"list", "migrations", "2"})
}

func TestCreateMigration_Sequential(t *testing.T) {
	dir := t.TempDir()
	prevDir := migrationsDir
	migrationsDir = dir
	t.Cleanup(func() { migrationsDir = prevDir })

	createMigration("first")
	createMigration("second")
	createMigration("third")

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	expected := []string{"001_first.sql", "002_second.sql", "003_third.sql"}
	for i, f := range files {
		if f.Name() != expected[i] {
			t.Errorf("file %d: expected %s, got %s", i, expected[i], f.Name())
		}
	}
}

func TestCreateMigration_FileContents(t *testing.T) {
	dir := t.TempDir()
	prevDir := migrationsDir
	migrationsDir = dir
	t.Cleanup(func() { migrationsDir = prevDir })

	createMigration("check_contents")

	data, err := os.ReadFile(filepath.Join(dir, "001_check_contents.sql"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "-- MIGRATE") {
		t.Error("missing -- MIGRATE section")
	}
	if !strings.Contains(content, "-- ROLLBACK") {
		t.Error("missing -- ROLLBACK section")
	}
}
