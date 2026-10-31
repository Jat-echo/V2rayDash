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
		`CREATE TABLE IF NOT EXISTS managed_nodes (
			server_id UUID PRIMARY KEY REFERENCES servers(id) ON DELETE CASCADE,
			role TEXT NOT NULL CHECK (role IN ('relay','exit')),
			layout TEXT NOT NULL CHECK (layout IN ('relay','exit','legacy')),
			inbound_tag TEXT NOT NULL,
			short_id TEXT NOT NULL DEFAULT '', egress_ip TEXT NOT NULL DEFAULT '',
			state TEXT NOT NULL DEFAULT 'adopted', last_error TEXT NOT NULL DEFAULT '',
			config_hash TEXT NOT NULL DEFAULT '', updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS relay_routes (
			account_id UUID PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
			exit_account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
			applied_exit_account_id UUID REFERENCES accounts(id) ON DELETE RESTRICT,
			revision BIGINT NOT NULL DEFAULT 1, applied_revision BIGINT NOT NULL DEFAULT 0,
			CHECK (account_id <> exit_account_id)
		)`,
		`ALTER TABLE accounts ADD COLUMN IF NOT EXISTS stats_key TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE accounts ADD COLUMN IF NOT EXISTS transport BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'exit' CHECK (role IN ('exit','relay'))`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS ssh_jump_server_id UUID REFERENCES servers(id) ON DELETE RESTRICT CHECK (ssh_jump_server_id<>id)`,
		`UPDATE servers s SET role='relay' FROM managed_nodes n WHERE n.server_id=s.id AND n.role='relay' AND s.role<>'relay'`,
		`CREATE TABLE IF NOT EXISTS relay_combinations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(100) NOT NULL UNIQUE,
			relay_server_id UUID NOT NULL REFERENCES servers(id) ON DELETE RESTRICT,
			exit_server_id UUID NOT NULL REFERENCES servers(id) ON DELETE RESTRICT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(relay_server_id,exit_server_id), CHECK(relay_server_id<>exit_server_id)
		)`,
		`ALTER TABLE accounts ADD COLUMN IF NOT EXISTS combination_id UUID REFERENCES relay_combinations(id) ON DELETE RESTRICT`,
		`ALTER TABLE relay_combinations ADD COLUMN IF NOT EXISTS exit_account_id UUID REFERENCES accounts(id) ON DELETE RESTRICT`,
		`DO $$ DECLARE c RECORD; upstream UUID; credential UUID; BEGIN
		 FOR c IN SELECT id,relay_server_id,exit_server_id FROM relay_combinations WHERE exit_account_id IS NULL FOR UPDATE LOOP
		  credential:=gen_random_uuid();
		  INSERT INTO accounts(server_id,uuid,email,protocols,stats_key,transport)
		   VALUES(c.exit_server_id,credential::text,'combination-'||c.id::text,ARRAY['vless_reality_vision'],'vd-'||credential::text,TRUE) RETURNING id INTO upstream;
		  UPDATE relay_combinations SET exit_account_id=upstream WHERE id=c.id;
		  UPDATE relay_routes SET exit_account_id=upstream,revision=revision+1 WHERE account_id IN(SELECT id FROM accounts WHERE combination_id=c.id);
		  UPDATE managed_nodes SET state='pending',last_error='',updated_at=CURRENT_TIMESTAMP WHERE server_id IN(c.relay_server_id,c.exit_server_id) AND state<>'publishing';
		 END LOOP; END $$`,
		`ALTER TABLE relay_combinations ALTER COLUMN exit_account_id SET NOT NULL`,
		`ALTER TABLE managed_nodes ADD COLUMN IF NOT EXISTS agent_token_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE managed_nodes ADD COLUMN IF NOT EXISTS ssh_host_key TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS agent_traffic_batches (
			server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
			batch_id UUID NOT NULL, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (server_id,batch_id)
		)`,
		`CREATE TABLE IF NOT EXISTS managed_account_state (
			account_id UUID PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
			uuid TEXT NOT NULL, enabled BOOLEAN NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS agent_traffic_cursors (
			server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
			account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			epoch TEXT NOT NULL, upload BIGINT NOT NULL DEFAULT 0, download BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY(server_id,account_id,epoch)
		)`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS remark TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS account_traffic_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			traffic_bytes BIGINT NOT NULL DEFAULT 0,
			recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_account_traffic_logs ON account_traffic_logs(account_id, recorded_at)`,
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'servers' AND column_name = 'country_code') THEN
				ALTER TABLE servers ADD COLUMN country_code VARCHAR(2) DEFAULT '';
			END IF;
		END $$`,
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
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'subscription_accounts' AND column_name = 'sort_order') THEN
				ALTER TABLE subscription_accounts ADD COLUMN sort_order INT DEFAULT 0;
			END IF;
		END $$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'subscription_accounts' AND column_name = 'note') THEN
				ALTER TABLE subscription_accounts ADD COLUMN note VARCHAR(255);
			END IF;
		END $$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'subscription_accounts' AND column_name = 'updated_at') THEN
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
			WHERE table_schema = current_schema() AND table_name = 'subscription_accounts' AND column_name = 'sort_order'),
		       EXISTS(SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = 'subscription_accounts' AND column_name = 'note')
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
