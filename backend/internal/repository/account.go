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

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(req *model.CreateAccountRequest) (*model.Account, error) {
	accountUUID := req.UUID
	if accountUUID == "" {
		accountUUID = uuid.New().String()
	}

	var id string
	err := r.db.QueryRow(
		`INSERT INTO accounts (server_id, uuid, email, protocols)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		req.ServerID, accountUUID, req.Email, pq.Array(req.Protocols),
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return r.GetByID(id)
}

func (r *AccountRepository) GetByID(id string) (*model.Account, error) {
	var a model.Account
	var protocols pq.StringArray
	err := r.db.QueryRow(
		`SELECT id, server_id, uuid, email, protocols, enabled, traffic_limit, traffic_used, created_at, updated_at
		 FROM accounts WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.ServerID, &a.UUID, &a.Email, &protocols, &a.Enabled, &a.TrafficLimit, &a.TrafficUsed, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	a.Protocols = protocols
	return &a, nil
}

func (r *AccountRepository) ListByServerID(serverID string) ([]*model.Account, error) {
	rows, err := r.db.Query(
		`SELECT id, server_id, uuid, email, protocols, enabled, traffic_limit, traffic_used, created_at, updated_at
		 FROM accounts WHERE server_id = $1 ORDER BY created_at DESC`,
		serverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var a model.Account
		var protocols pq.StringArray
		if err := rows.Scan(&a.ID, &a.ServerID, &a.UUID, &a.Email, &protocols, &a.Enabled, &a.TrafficLimit, &a.TrafficUsed, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.Protocols = protocols
		accounts = append(accounts, &a)
	}
	return accounts, nil
}

func (r *AccountRepository) List() ([]*model.Account, error) {
	rows, err := r.db.Query(
		`SELECT id, server_id, uuid, email, protocols, enabled, traffic_limit, traffic_used, created_at, updated_at
		 FROM accounts ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var a model.Account
		var protocols pq.StringArray
		if err := rows.Scan(&a.ID, &a.ServerID, &a.UUID, &a.Email, &protocols, &a.Enabled, &a.TrafficLimit, &a.TrafficUsed, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.Protocols = protocols
		accounts = append(accounts, &a)
	}
	return accounts, nil
}

func (r *AccountRepository) Update(id string, req *model.UpdateAccountRequest) error {
	var setClauses []string
	var args []interface{}
	argNum := 1

	if req.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argNum))
		args = append(args, *req.Email)
		argNum++
	}
	if req.Protocols != nil {
		setClauses = append(setClauses, fmt.Sprintf("protocols = $%d", argNum))
		args = append(args, pq.Array(req.Protocols))
		argNum++
	}
	if req.Enabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("enabled = $%d", argNum))
		args = append(args, *req.Enabled)
		argNum++
	}
	if req.TrafficLimit != nil {
		setClauses = append(setClauses, fmt.Sprintf("traffic_limit = $%d", argNum))
		args = append(args, *req.TrafficLimit)
		argNum++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	args = append(args, id)

	query := fmt.Sprintf("UPDATE accounts SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argNum)
	_, err := r.db.Exec(query, args...)
	return err
}

func (r *AccountRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM accounts WHERE id = $1", id)
	return err
}