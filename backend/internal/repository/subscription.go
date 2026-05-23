package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"v2ray-dash/backend/internal/model"
)

type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(req *model.CreateSubscriptionRequest) (*model.Subscription, error) {
	subUUID := generateUUID()

	// 获取第一个账号关联的 server_id
	var serverID string
	for _, mapping := range req.AccountMappings {
		if mapping.AccountID != "" {
			var sid string
			err := r.db.QueryRow(`SELECT server_id FROM accounts WHERE id = $1`, mapping.AccountID).Scan(&sid)
			if err == nil {
				serverID = sid
				break
			}
		}
	}

	var s model.Subscription
	err := r.db.QueryRow(
		`INSERT INTO subscriptions (name, uuid, traffic_limit, server_id)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, server_id, name, uuid, enable, traffic_limit, traffic_used, created_at, updated_at`,
		req.Name, subUUID, req.TrafficLimit, serverID,
	).Scan(&s.ID, &s.ServerID, &s.Name, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SubscriptionRepository) GetByID(id string) (*model.Subscription, error) {
	var s model.Subscription
	err := r.db.QueryRow(
		`SELECT id, server_id, name, uuid, enable, traffic_limit, traffic_used, created_at, updated_at
		 FROM subscriptions WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.ServerID, &s.Name, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt)
	return &s, err
}

func (r *SubscriptionRepository) GetByUUID(uuid string) (*model.Subscription, error) {
	var s model.Subscription
	err := r.db.QueryRow(
		`SELECT id, server_id, name, uuid, enable, traffic_limit, traffic_used, created_at, updated_at
		 FROM subscriptions WHERE uuid = $1`,
		uuid,
	).Scan(&s.ID, &s.ServerID, &s.Name, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt)
	return &s, err
}

func (r *SubscriptionRepository) List() ([]*model.Subscription, error) {
	rows, err := r.db.Query(
		`SELECT id, server_id, name, uuid, enable, traffic_limit, traffic_used, created_at, updated_at
		 FROM subscriptions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		var s model.Subscription
		if err := rows.Scan(&s.ID, &s.ServerID, &s.Name, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, &s)
	}
	if subs == nil {
		subs = []*model.Subscription{}
	}
	return subs, nil
}

func (r *SubscriptionRepository) ListByServerID(serverID string) ([]*model.Subscription, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT s.id, s.name, s.uuid, s.enable, s.traffic_limit, s.traffic_used, s.created_at, s.updated_at
		FROM subscriptions s
		JOIN subscription_accounts sa ON s.id = sa.subscription_id
		JOIN accounts a ON sa.account_id = a.id
		WHERE a.server_id = $1
		ORDER BY s.created_at DESC`,
		serverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		var s model.Subscription
		if err := rows.Scan(&s.ID, &s.Name, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, &s)
	}
	if subs == nil {
		subs = []*model.Subscription{}
	}
	return subs, nil
}

func (r *SubscriptionRepository) Update(id string, req *model.UpdateSubscriptionRequest) error {
	_, err := r.db.Exec(`UPDATE subscriptions SET updated_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

func (r *SubscriptionRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM subscriptions WHERE id = $1`, id)
	return err
}

func (r *SubscriptionRepository) GetAccountIDs(subscriptionID string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT account_id FROM subscription_accounts WHERE subscription_id = $1
	`, subscriptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

func (r *SubscriptionRepository) GetSubscriptionsWithAccounts() ([]*model.SubscriptionWithAccounts, error) {
	rows, err := r.db.Query(`
		SELECT s.id, s.server_id, s.name, s.uuid, s.enable, s.traffic_limit, s.traffic_used, s.created_at, s.updated_at,
		       a.id, a.server_id, a.uuid, a.email, a.protocols, a.enabled,
		       a.traffic_limit, a.traffic_used, a.created_at, a.updated_at,
		       srv.name, srv.ip
		FROM subscriptions s
		LEFT JOIN subscription_accounts sa ON s.id = sa.subscription_id
		LEFT JOIN accounts a ON sa.account_id = a.id
		LEFT JOIN servers srv ON a.server_id = srv.id
		ORDER BY s.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subMap := make(map[string]*model.SubscriptionWithAccounts)
	var subs []*model.SubscriptionWithAccounts

	for rows.Next() {
		var subID, subServerID, subName, subUUID string
		var subEnable bool
		var subTrafficLimit, subTrafficUsed int64
		var subCreated, subUpdated time.Time
		var accID, accServerID, accUUID, accEmail sql.NullString
		var accProtocols pq.StringArray
		var accEnabled sql.NullBool
		var accTrafficLimit, accTrafficUsed sql.NullInt64
		var accCreated, accUpdated sql.NullTime
		var serverName, serverIP sql.NullString

		err := rows.Scan(
			&subID, &subServerID, &subName, &subUUID, &subEnable, &subTrafficLimit, &subTrafficUsed, &subCreated, &subUpdated,
			&accID, &accServerID, &accUUID, &accEmail, &accProtocols, &accEnabled,
			&accTrafficLimit, &accTrafficUsed, &accCreated, &accUpdated,
			&serverName, &serverIP,
		)
		if err != nil {
			return nil, err
		}

		if _, exists := subMap[subID]; !exists {
			subMap[subID] = &model.SubscriptionWithAccounts{
				Subscription: model.Subscription{
					ID:           subID,
					ServerID:     subServerID,
					Name:         subName,
					UUID:         subUUID,
					Enable:       subEnable,
					TrafficLimit: subTrafficLimit,
					TrafficUsed:  subTrafficUsed,
					CreatedAt:    subCreated,
					UpdatedAt:    subUpdated,
				},
				Accounts: []*model.AccountWithServerInfo{},
			}
		}

		if accID.Valid {
			acc := &model.AccountWithServerInfo{
				Account: model.Account{
					ID:           accID.String,
					ServerID:     accServerID.String,
					UUID:         accUUID.String,
					Email:        accEmail.String,
					Protocols:    accProtocols,
					Enabled:      accEnabled.Bool,
					TrafficLimit: accTrafficLimit.Int64,
					TrafficUsed:  accTrafficUsed.Int64,
					CreatedAt:    accCreated.Time,
					UpdatedAt:    accUpdated.Time,
				},
				ServerName: serverName.String,
				ServerIP:   serverIP.String,
			}
			subMap[subID].Accounts = append(subMap[subID].Accounts, acc)
		}
	}

	for _, sub := range subMap {
		subs = append(subs, sub)
	}
	if subs == nil {
		subs = []*model.SubscriptionWithAccounts{}
	}
	return subs, nil
}

func (r *SubscriptionRepository) GetByIDWithAccounts(id string) (*model.SubscriptionWithAccounts, error) {
	row := r.db.QueryRow(`
		SELECT s.id, s.server_id, s.name, s.uuid, s.enable, s.traffic_limit, s.traffic_used, s.created_at, s.updated_at,
		       a.id, a.server_id, a.uuid, a.email, a.protocols, a.enabled,
		       a.traffic_limit, a.traffic_used, a.created_at, a.updated_at,
		       srv.name, srv.ip
		FROM subscriptions s
		LEFT JOIN subscription_accounts sa ON s.id = sa.subscription_id
		LEFT JOIN accounts a ON sa.account_id = a.id
		LEFT JOIN servers srv ON a.server_id = srv.id
		WHERE s.id = $1
	`, id)

	var subID, subServerID, subName, subUUID string
	var subEnable bool
	var subTrafficLimit, subTrafficUsed int64
	var subCreated, subUpdated time.Time
	var accID, accServerID, accUUID, accEmail sql.NullString
	var accProtocols pq.StringArray
	var accEnabled sql.NullBool
	var accTrafficLimit, accTrafficUsed sql.NullInt64
	var accCreated, accUpdated sql.NullTime
	var serverName, serverIP sql.NullString

	err := row.Scan(
		&subID, &subServerID, &subName, &subUUID, &subEnable, &subTrafficLimit, &subTrafficUsed, &subCreated, &subUpdated,
		&accID, &accServerID, &accUUID, &accEmail, &accProtocols, &accEnabled,
		&accTrafficLimit, &accTrafficUsed, &accCreated, &accUpdated,
		&serverName, &serverIP,
	)
	if err != nil {
		return nil, err
	}

	result := &model.SubscriptionWithAccounts{
		Subscription: model.Subscription{
			ID:           subID,
			ServerID:     subServerID,
			Name:         subName,
			UUID:         subUUID,
			Enable:       subEnable,
			TrafficLimit: subTrafficLimit,
			TrafficUsed:  subTrafficUsed,
			CreatedAt:    subCreated,
			UpdatedAt:    subUpdated,
		},
		Accounts: []*model.AccountWithServerInfo{},
	}

	if accID.Valid {
		result.Accounts = append(result.Accounts, &model.AccountWithServerInfo{
			Account: model.Account{
				ID:           accID.String,
				ServerID:     accServerID.String,
				UUID:         accUUID.String,
				Email:        accEmail.String,
				Protocols:    accProtocols,
				Enabled:      accEnabled.Bool,
				TrafficLimit: accTrafficLimit.Int64,
				TrafficUsed:  accTrafficUsed.Int64,
				CreatedAt:    accCreated.Time,
				UpdatedAt:    accUpdated.Time,
			},
			ServerName: serverName.String,
			ServerIP:   serverIP.String,
		})
	}

	return result, nil
}

func generateUUID() string {
	return uuid.New().String()
}