package repository

import (
	"database/sql"
	"encoding/json"
	"time"

	"v2ray-dash/backend/internal/model"
)

type ServerRepository struct {
	db *sql.DB
}

func NewServerRepository(db *sql.DB) *ServerRepository {
	return &ServerRepository{db: db}
}

func (r *ServerRepository) Create(req *model.CreateServerRequest) (*model.Server, error) {
	tagsJSON, _ := json.Marshal(req.Tags)
	sshKeyType := req.SSHKeyType
	if sshKeyType == "" {
		sshKeyType = "key" // default to key-based auth
	}
	result := r.db.QueryRow(
		`INSERT INTO servers (name, ip, ssh_port, ssh_user, ssh_key_type, ssh_key, ssh_password, tags)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, name, ip, ssh_port, ssh_user, ssh_key_type, tags, status, created_at, updated_at`,
		req.Name, req.IP, req.SSHPort, req.SSHUser, sshKeyType, req.SSHKey, req.SSHPassword, tagsJSON,
	)

	var s model.Server
	var tagsBytes []byte
	err := result.Scan(&s.ID, &s.Name, &s.IP, &s.SSHPort, &s.SSHUser, &s.SSHKeyType, &tagsBytes, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tagsBytes, &s.Tags)
	return &s, nil
}

func (r *ServerRepository) GetByID(id string) (*model.Server, error) {
	var s model.Server
	var tagsBytes []byte
	err := r.db.QueryRow(
		`SELECT id, name, ip, ssh_port, ssh_user, ssh_key_type, tags, status, created_at, updated_at
		 FROM servers WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.Name, &s.IP, &s.SSHPort, &s.SSHUser, &s.SSHKeyType, &tagsBytes, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tagsBytes, &s.Tags)
	return &s, nil
}

func (r *ServerRepository) List() ([]*model.Server, error) {
	rows, err := r.db.Query(
		`SELECT id, name, ip, ssh_port, ssh_user, ssh_key_type, tags, status, created_at, updated_at
		 FROM servers ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*model.Server
	for rows.Next() {
		var s model.Server
		var tagsBytes []byte
		if err := rows.Scan(&s.ID, &s.Name, &s.IP, &s.SSHPort, &s.SSHUser, &s.SSHKeyType, &tagsBytes, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(tagsBytes, &s.Tags)
		servers = append(servers, &s)
	}
	if servers == nil {
		servers = []*model.Server{}
	}
	return servers, nil
}

func (r *ServerRepository) Update(id string, req *model.UpdateServerRequest) (*model.Server, error) {
	// 实现动态更新逻辑
	return r.GetByID(id)
}

func (r *ServerRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM servers WHERE id = $1`, id)
	return err
}

func (r *ServerRepository) UpdateStatus(id, status string) error {
	_, err := r.db.Exec(
		`UPDATE servers SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now(), id,
	)
	return err
}