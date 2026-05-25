package main

import (
	"os"
	"testing"
)

func TestConnectDB_CreatesAndAttaches(t *testing.T) {
	prev := dbFile
	t.Cleanup(func() {
		dbFile = prev
		os.Remove("test_connectdb.db")
	})
	dbFile = "test_connectdb.db"

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB() error = %v", err)
	}
	defer db.Close()

	var n int
	if err = db.QueryRow("SELECT 42").Scan(&n); err != nil {
		t.Fatalf("query after connectDB failed: %v", err)
	}
	if n != 42 {
		t.Fatalf("expected 42, got %d", n)
	}
}

func TestConnectDB_RespectsEncKey(t *testing.T) {
	prev := dbFile
	t.Cleanup(func() {
		dbFile = prev
		os.Remove("test_enc.db")
	})
	dbFile = "test_enc.db"
	t.Setenv("ENC_KEY", "supersecret")

	db, err := connectDB()
	if err != nil {
		t.Fatalf("connectDB() with ENC_KEY error = %v", err)
	}
	db.Close()
}

func TestIsSyncTableInitialized_ReturnsFalseOnBadDB(t *testing.T) {
	prev := dbFile
	t.Cleanup(func() { dbFile = prev })
	// Point at a path that cannot be created (invalid characters)
	dbFile = string([]byte{0})

	result := isSyncTableInitialized()
	if result {
		t.Error("expected false when DB cannot be opened, got true")
	}
}

func TestIsSyncTableInitialized_ReturnsTrueWhenAccessible(t *testing.T) {
	prev := dbFile
	t.Cleanup(func() {
		dbFile = prev
		os.Remove("test_syncinit.db")
	})
	dbFile = "test_syncinit.db"

	if !isSyncTableInitialized() {
		t.Error("expected true when DB is accessible, got false")
	}
}
