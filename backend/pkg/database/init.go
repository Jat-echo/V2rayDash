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
	// Migration: Add sort_order, note, updated_at columns to subscription_accounts if they don't exist
	// Using proper PostgreSQL DO blocks to check column existence before adding
	migrations := []string{
		// 管理员用户表
		`CREATE TABLE IF NOT EXISTS admin_users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(50) NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		// 订阅流量日志表（用于时间序列流量图表）
		`CREATE TABLE IF NOT EXISTS subscription_traffic_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
			traffic_bytes BIGINT NOT NULL DEFAULT 0,
			recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sub_traffic_logs ON subscription_traffic_logs(subscription_id, recorded_at)`,
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'subscription_accounts' AND column_name = 'sort_order') THEN
				ALTER TABLE subscription_accounts ADD COLUMN sort_order INT DEFAULT 0;
			END IF;
		END $$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'subscription_accounts' AND column_name = 'note') THEN
				ALTER TABLE subscription_accounts ADD COLUMN note VARCHAR(255);
			END IF;
		END $$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'subscription_accounts' AND column_name = 'updated_at') THEN
				ALTER TABLE subscription_accounts ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
			END IF;
		END $$`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Check if migration is already completed by verifying columns exist
	var sortOrderExists, noteExists bool
	err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM information_schema.columns
			WHERE table_name = 'subscription_accounts' AND column_name = 'sort_order'),
		       EXISTS(SELECT 1 FROM information_schema.columns
			WHERE table_name = 'subscription_accounts' AND column_name = 'note')
	`).Scan(&sortOrderExists, &noteExists)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if sortOrderExists && noteExists {
		fmt.Println("Migration already completed, skipping")
		return nil
	}

	var subCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM subscriptions`).Scan(&subCount)
	if err != nil {
		return fmt.Errorf("failed to check subscriptions: %w", err)
	}
	if subCount == 0 {
		return nil
	}

	fmt.Println("Running migration: migrating subscription data")
	if _, err := db.Exec(migrationSQL); err != nil {
		return fmt.Errorf("failed to run migration: %w", err)
	}

	return nil
}

func RunMigrationManually(db *sql.DB) error {
	return runMigrations(&DB{db})
}