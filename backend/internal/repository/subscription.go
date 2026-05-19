package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"v2ray-dash/backend/internal/model"
)

type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(req *model.CreateSubscriptionRequest) (*model.Subscription, error) {
	uuid := generateUUID()
	result := r.db.QueryRow(
		`INSERT INTO subscriptions (server_id, name, uuid, traffic_limit)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, server_id, name, uuid, enable, traffic_limit, traffic_used, created_at, updated_at`,
		req.ServerID, req.Name, uuid, req.TrafficLimit,
	)

	var s model.Subscription
	err := result.Scan(&s.ID, &s.ServerID, &s.Name, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt)
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
	rows, err := r.db.Query(
		`SELECT id, server_id, name, uuid, enable, traffic_limit, traffic_used, created_at, updated_at
		 FROM subscriptions WHERE server_id = $1 ORDER BY created_at DESC`,
		serverID,
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

func (r *SubscriptionRepository) Update(id string, req *model.UpdateSubscriptionRequest) error {
	// 实现动态更新
	_, err := r.db.Exec(`UPDATE subscriptions SET updated_at = $1 WHERE id = $2`, time.Now(), id)
	return err
}

func (r *SubscriptionRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM subscriptions WHERE id = $1`, id)
	return err
}

func generateUUID() string {
	return uuid.New().String()
}