package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: No .env file found or failed to load .env file.")
	}

	flag.StringVar(&dbFile, "db", "duckdb", "Database file (default 'duckdb')")
	flag.Parse()

	if dbFile == "duckdb" {
		if dbEnv := os.Getenv("DATABASE"); dbEnv != "" {
			dbFile = dbEnv
		}
	}

	if len(flag.Args()) < 1 {
		fmt.Println("Usage: duckdbm [init|create|apply|rollback|list|sync|validate] [options]")
		return
	}

	switch flag.Args()[0] {
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
		n := 1
		if len(flag.Args()) > 1 {
			var err error
			n, err = strconv.Atoi(flag.Args()[1])
			if err != nil || n <= 0 {
				fmt.Println("Please provide a valid positive number for rollback count.")
				return
			}
		}
		rollbackLast(n)
	case "list":
		listAppliedMigrations(os.Args[2:])
	case "sync":
		if len(flag.Args()) < 2 {
			fmt.Println("Please provide the name of the migration to sync.")
			return
		}
		syncMigration(flag.Args()[1])
	case "validate":
		validateMigrations(flag.Args(), migrationsDir)
		validateMigrations(flag.Args(), migrationsDir+"/sync")
	default:
		fmt.Printf("Unknown command: %s\n", flag.Args()[0])
	}
}
