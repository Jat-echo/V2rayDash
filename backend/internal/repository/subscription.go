package repository

import (
	"database/sql"
	"fmt"
	"strings"
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
		`INSERT INTO subscriptions (name, remark, uuid, traffic_limit, server_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, server_id, name, remark, uuid, enable, traffic_limit, traffic_used, created_at, updated_at`,
		req.Name, req.Remark, subUUID, req.TrafficLimit, serverID,
	).Scan(&s.ID, &s.ServerID, &s.Name, &s.Remark, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SubscriptionRepository) GetByID(id string) (*model.Subscription, error) {
	var s model.Subscription
	err := r.db.QueryRow(
		`SELECT id, COALESCE(server_id::text, ''), name, remark, uuid, enable, traffic_limit, traffic_used, created_at, updated_at
		 FROM subscriptions WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.ServerID, &s.Name, &s.Remark, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt)
	return &s, err
}

func (r *SubscriptionRepository) GetByUUID(uuid string) (*model.Subscription, error) {
	var s model.Subscription
	err := r.db.QueryRow(
		`SELECT id, COALESCE(server_id::text, ''), name, remark, uuid, enable, traffic_limit, traffic_used, created_at, updated_at
		 FROM subscriptions WHERE uuid = $1`,
		uuid,
	).Scan(&s.ID, &s.ServerID, &s.Name, &s.Remark, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt)
	return &s, err
}

func (r *SubscriptionRepository) List() ([]*model.Subscription, error) {
	rows, err := r.db.Query(
		`SELECT id, COALESCE(server_id::text, ''), name, remark, uuid, enable, traffic_limit, traffic_used, created_at, updated_at
		 FROM subscriptions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		var s model.Subscription
		if err := rows.Scan(&s.ID, &s.ServerID, &s.Name, &s.Remark, &s.UUID, &s.Enable, &s.TrafficLimit, &s.TrafficUsed, &s.CreatedAt, &s.UpdatedAt); err != nil {
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
	setClauses := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []interface{}{}
	idx := 1
	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", idx))
		args = append(args, *req.Name)
		idx++
	}
	if req.Remark != nil {
		setClauses = append(setClauses, fmt.Sprintf("remark = $%d", idx))
		args = append(args, *req.Remark)
		idx++
	}
	if req.Enable != nil {
		setClauses = append(setClauses, fmt.Sprintf("enable = $%d", idx))
		args = append(args, *req.Enable)
		idx++
	}
	if req.TrafficLimit != nil {
		setClauses = append(setClauses, fmt.Sprintf("traffic_limit = $%d", idx))
		args = append(args, *req.TrafficLimit)
		idx++
	}
	args = append(args, id)
	query := fmt.Sprintf("UPDATE subscriptions SET %s WHERE id = $%d", strings.Join(setClauses, ", "), idx)
	_, err := r.db.Exec(query, args...)
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
		SELECT s.id, COALESCE(s.server_id::text, ''), s.name, s.remark, s.uuid, s.enable, s.traffic_limit, s.traffic_used, s.created_at, s.updated_at,
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
		var subID, subServerID, subName, subRemark, subUUID string
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
			&subID, &subServerID, &subName, &subRemark, &subUUID, &subEnable, &subTrafficLimit, &subTrafficUsed, &subCreated, &subUpdated,
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
					Remark:       subRemark,
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
		SELECT s.id, COALESCE(s.server_id::text, ''), s.name, s.remark, s.uuid, s.enable, s.traffic_limit, s.traffic_used, s.created_at, s.updated_at,
		       a.id, a.server_id, a.uuid, a.email, a.protocols, a.enabled,
		       a.traffic_limit, a.traffic_used, a.created_at, a.updated_at,
		       srv.name, srv.ip
		FROM subscriptions s
		LEFT JOIN subscription_accounts sa ON s.id = sa.subscription_id
		LEFT JOIN accounts a ON sa.account_id = a.id
		LEFT JOIN servers srv ON a.server_id = srv.id
		WHERE s.id = $1
	`, id)

	var subID, subServerID, subName, subRemark, subUUID string
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
		&subID, &subServerID, &subName, &subRemark, &subUUID, &subEnable, &subTrafficLimit, &subTrafficUsed, &subCreated, &subUpdated,
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
			Remark:       subRemark,
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

// GetByAccountID returns all subscription IDs that contain the given account
func (r *SubscriptionRepository) GetByAccountID(accountID string) ([]string, error) {
	rows, err := r.db.Query(
		`SELECT subscription_id FROM subscription_accounts WHERE account_id = $1`,
		accountID,
	)
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
	return ids, nil
}

// RecalcTrafficUsed recalculates subscription.traffic_used as the sum of its accounts' traffic_used
func (r *SubscriptionRepository) RecalcTrafficUsed(subscriptionID string) error {
	_, err := r.db.Exec(`
		UPDATE subscriptions SET
			traffic_used = (
				SELECT COALESCE(SUM(a.traffic_used), 0)
				FROM subscription_accounts sa
				JOIN accounts a ON a.id = sa.account_id
				WHERE sa.subscription_id = $1
			),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, subscriptionID)
	return err
}

// LogTraffic inserts a traffic snapshot for time-series charting
func (r *SubscriptionRepository) LogTraffic(subscriptionID string, trafficBytes int64) error {
	_, err := r.db.Exec(
		`INSERT INTO subscription_traffic_logs (subscription_id, traffic_bytes) VALUES ($1, $2)`,
		subscriptionID, trafficBytes,
	)
	return err
}

// GetAccountTrafficLogs returns per-account cumulative traffic snapshots for a subscription,
// bucketed by time range to limit returned point count.
func (r *SubscriptionRepository) GetAccountTrafficLogs(subscriptionID, timeRange string) ([]model.AccountTrafficSeries, error) {
	interval := timeRangeToInterval(timeRange, "1 day")
	bucket := timeRangeToBucketSeconds(timeRange)

	rows, err := r.db.Query(`
		SELECT
		  a.id,
		  a.email,
		  srv.name,
		  MAX(atl.traffic_bytes) AS traffic_bytes,
		  to_timestamp(floor(extract(epoch from atl.recorded_at) / $3) * $3) AS bucket_time
		FROM subscription_accounts sa
		JOIN accounts a ON sa.account_id = a.id
		JOIN servers srv ON a.server_id = srv.id
		JOIN account_traffic_logs atl ON atl.account_id = a.id
		WHERE sa.subscription_id = $1
		  AND atl.recorded_at > NOW() - $2::interval
		GROUP BY a.id, a.email, srv.name,
		         to_timestamp(floor(extract(epoch from atl.recorded_at) / $3) * $3)
		ORDER BY a.id, bucket_time ASC
	`, subscriptionID, interval, bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seriesMap := make(map[string]*model.AccountTrafficSeries)
	var order []string
	for rows.Next() {
		var accID, email, serverName string
		var bytes int64
		var t time.Time
		if err := rows.Scan(&accID, &email, &serverName, &bytes, &t); err != nil {
			return nil, err
		}
		if _, ok := seriesMap[accID]; !ok {
			seriesMap[accID] = &model.AccountTrafficSeries{
				AccountID:  accID,
				Email:      email,
				ServerName: serverName,
				Points:     []model.BandwidthPoint{},
			}
			order = append(order, accID)
		}
		seriesMap[accID].Points = append(seriesMap[accID].Points, model.BandwidthPoint{Time: t, Value: bytes})
	}

	result := make([]model.AccountTrafficSeries, 0, len(order))
	for _, id := range order {
		result = append(result, *seriesMap[id])
	}
	return result, nil
}

// GetTrafficLogs returns traffic snapshots within the given time range (e.g. "1h", "1d", "7d"),
// bucketed by time range to limit returned point count.
func (r *SubscriptionRepository) GetTrafficLogs(subscriptionID, timeRange string) ([]model.BandwidthPoint, error) {
	interval := timeRangeToInterval(timeRange, "1 day")
	bucket := timeRangeToBucketSeconds(timeRange)
	rows, err := r.db.Query(`
		SELECT
		  MAX(traffic_bytes) AS traffic_bytes,
		  to_timestamp(floor(extract(epoch from recorded_at) / $3) * $3) AS bucket_time
		FROM subscription_traffic_logs
		WHERE subscription_id = $1 AND recorded_at > NOW() - $2::interval
		GROUP BY to_timestamp(floor(extract(epoch from recorded_at) / $3) * $3)
		ORDER BY bucket_time ASC`, subscriptionID, interval, bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var points []model.BandwidthPoint
	for rows.Next() {
		var p model.BandwidthPoint
		if err := rows.Scan(&p.Value, &p.Time); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, nil
}