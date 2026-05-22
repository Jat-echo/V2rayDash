package database

import (
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schemaSQL string

//go:embed migration_add_subscription_accounts.sql
var migrationSQL string

func InitSchema(db *DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func runMigrations(db *DB) error {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM subscription_accounts`).Scan(&count)
	if err != nil {
		return nil
	}

	if count > 0 {
		fmt.Println("Migration already completed, skipping")
		return nil
	}

	var subCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM subscriptions`).Scan(&subCount)
	if err != nil || subCount == 0 {
		return nil
	}

	fmt.Println("Running migration: migrating subscription data")
	if _, err := db.Exec(migrationSQL); err != nil {
		fmt.Printf("Migration warning: %v\n", err)
	}

	return nil
}

func RunMigrationManually(db *sql.DB) error {
	return runMigrations(&DB{db})
}