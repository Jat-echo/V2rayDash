package repository

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"v2ray-dash/backend/internal/model"
)

type SubscriptionAccountRepository struct {
	db *sql.DB
}

func NewSubscriptionAccountRepository(db *sql.DB) *SubscriptionAccountRepository {
	return &SubscriptionAccountRepository{db: db}
}

func (r *SubscriptionAccountRepository) AddAccount(subscriptionID, accountID string) error {
	existingCount := 0
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM subscription_accounts sa
		JOIN accounts a ON sa.account_id = a.id
		WHERE sa.subscription_id = $1 AND a.server_id = (
			SELECT server_id FROM accounts WHERE id = $2
		)
	`, subscriptionID, accountID).Scan(&existingCount)
	if err != nil {
		return err
	}
	if existingCount > 0 {
		return fmt.Errorf("该订阅已选择此服务器的其他账号")
	}

	_, err = r.db.Exec(`
		INSERT INTO subscription_accounts (subscription_id, account_id)
		VALUES ($1, $2)
	`, subscriptionID, accountID)
	return err
}

func (r *SubscriptionAccountRepository) RemoveAccount(subscriptionID, accountID string) error {
	_, err := r.db.Exec(`
		DELETE FROM subscription_accounts
		WHERE subscription_id = $1 AND account_id = $2
	`, subscriptionID, accountID)
	return err
}

func (r *SubscriptionAccountRepository) ListBySubscriptionID(subscriptionID string) ([]*model.Account, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.server_id, a.uuid, a.email, a.protocols, a.enabled,
		       a.traffic_limit, a.traffic_used, a.created_at, a.updated_at
		FROM accounts a
		JOIN subscription_accounts sa ON a.id = sa.account_id
		WHERE sa.subscription_id = $1
		ORDER BY a.created_at DESC
	`, subscriptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var a model.Account
		var protocols pq.StringArray
		if err := rows.Scan(&a.ID, &a.ServerID, &a.UUID, &a.Email, &protocols,
			&a.Enabled, &a.TrafficLimit, &a.TrafficUsed, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.Protocols = protocols
		accounts = append(accounts, &a)
	}
	if accounts == nil {
		accounts = []*model.Account{}
	}
	return accounts, nil
}

func (r *SubscriptionAccountRepository) GetAccountsWithServerInfo(subscriptionID string) ([]*model.AccountWithServerInfo, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.server_id, a.uuid, a.email, a.protocols, a.enabled,
		       a.traffic_limit, a.traffic_used, a.created_at, a.updated_at,
		       s.name as server_name, s.ip as server_ip
		FROM accounts a
		JOIN subscription_accounts sa ON a.id = sa.account_id
		JOIN servers s ON a.server_id = s.id
		WHERE sa.subscription_id = $1
		ORDER BY a.created_at DESC
	`, subscriptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.AccountWithServerInfo
	for rows.Next() {
		var a model.AccountWithServerInfo
		var protocols pq.StringArray
		if err := rows.Scan(&a.ID, &a.ServerID, &a.UUID, &a.Email, &protocols,
			&a.Enabled, &a.TrafficLimit, &a.TrafficUsed, &a.CreatedAt, &a.UpdatedAt,
			&a.ServerName, &a.ServerIP); err != nil {
			return nil, err
		}
		a.Protocols = protocols
		accounts = append(accounts, &a)
	}
	if accounts == nil {
		accounts = []*model.AccountWithServerInfo{}
	}
	return accounts, nil
}

func (r *SubscriptionAccountRepository) DeleteBySubscription(subscriptionID string) error {
	_, err := r.db.Exec(`DELETE FROM subscription_accounts WHERE subscription_id = $1`, subscriptionID)
	return err
}

func (r *SubscriptionAccountRepository) ReplaceAccounts(subscriptionID string, accountIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM subscription_accounts WHERE subscription_id = $1`, subscriptionID)
	if err != nil {
		return err
	}

	for _, accountID := range accountIDs {
		_, err = tx.Exec(`
			INSERT INTO subscription_accounts (subscription_id, account_id)
			VALUES ($1, $2)
		`, subscriptionID, accountID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}