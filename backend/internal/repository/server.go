package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
		`INSERT INTO servers (name, ip, ssh_port, ssh_user, ssh_key_type, ssh_key, ssh_password, tags, reality_enabled, reality_server_name, reality_public_key, reality_port)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id, name, ip, ssh_port, ssh_user, ssh_key_type, tags, status, reality_enabled, reality_server_name, reality_public_key, reality_port, created_at, updated_at`,
		req.Name, req.IP, req.SSHPort, req.SSHUser, sshKeyType, req.SSHKey, req.SSHPassword, tagsJSON, req.RealityEnabled, req.RealityServerName, req.RealityPublicKey, req.RealityPort,
	)

	var s model.Server
	var tagsBytes []byte
	err := result.Scan(&s.ID, &s.Name, &s.IP, &s.SSHPort, &s.SSHUser, &s.SSHKeyType, &tagsBytes, &s.Status, &s.RealityEnabled, &s.RealityServerName, &s.RealityPublicKey, &s.RealityPort, &s.CreatedAt, &s.UpdatedAt)
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
		`SELECT id, name, ip, ssh_port, ssh_user, ssh_key_type, tags, status, reality_enabled, reality_server_name, reality_public_key, reality_port, created_at, updated_at
		 FROM servers WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.Name, &s.IP, &s.SSHPort, &s.SSHUser, &s.SSHKeyType, &tagsBytes, &s.Status, &s.RealityEnabled, &s.RealityServerName, &s.RealityPublicKey, &s.RealityPort, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tagsBytes, &s.Tags)
	return &s, nil
}

// GetByIDForInstall 获取服务器完整信息（包括敏感字段），仅供内部使用
func (r *ServerRepository) GetByIDForInstall(id string) (*model.Server, error) {
	var s model.Server
	var tagsBytes []byte
	err := r.db.QueryRow(
		`SELECT id, name, ip, ssh_port, ssh_user, ssh_key_type, ssh_key, ssh_password, tags, status, reality_enabled, reality_server_name, reality_public_key, reality_port, created_at, updated_at
		 FROM servers WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.Name, &s.IP, &s.SSHPort, &s.SSHUser, &s.SSHKeyType, &s.SSHKey, &s.SSHPassword, &tagsBytes, &s.Status, &s.RealityEnabled, &s.RealityServerName, &s.RealityPublicKey, &s.RealityPort, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tagsBytes, &s.Tags)
	return &s, nil
}

func (r *ServerRepository) List() ([]*model.Server, error) {
	rows, err := r.db.Query(
		`SELECT id, name, ip, ssh_port, ssh_user, ssh_key_type, tags, status, reality_enabled, reality_server_name, reality_public_key, reality_port, created_at, updated_at
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
		if err := rows.Scan(&s.ID, &s.Name, &s.IP, &s.SSHPort, &s.SSHUser, &s.SSHKeyType, &tagsBytes, &s.Status, &s.RealityEnabled, &s.RealityServerName, &s.RealityPublicKey, &s.RealityPort, &s.CreatedAt, &s.UpdatedAt); err != nil {
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
	// Build dynamic update query
	setClauses := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *req.Name)
		argNum++
	}
	if req.SSHPort != nil {
		setClauses = append(setClauses, fmt.Sprintf("ssh_port = $%d", argNum))
		args = append(args, *req.SSHPort)
		argNum++
	}
	if req.SSHUser != nil {
		setClauses = append(setClauses, fmt.Sprintf("ssh_user = $%d", argNum))
		args = append(args, *req.SSHUser)
		argNum++
	}
	if req.SSHKeyType != nil {
		setClauses = append(setClauses, fmt.Sprintf("ssh_key_type = $%d", argNum))
		args = append(args, *req.SSHKeyType)
		argNum++
	}
	if req.SSHKey != nil {
		setClauses = append(setClauses, fmt.Sprintf("ssh_key = $%d", argNum))
		args = append(args, *req.SSHKey)
		argNum++
	}
	if req.SSHPassword != nil {
		setClauses = append(setClauses, fmt.Sprintf("ssh_password = $%d", argNum))
		args = append(args, *req.SSHPassword)
		argNum++
	}
	if req.Tags != nil {
		tagsJSON, _ := json.Marshal(req.Tags)
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argNum))
		args = append(args, tagsJSON)
		argNum++
	}
	if req.RealityEnabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("reality_enabled = $%d", argNum))
		args = append(args, *req.RealityEnabled)
		argNum++
	}
	if req.RealityServerName != nil {
		setClauses = append(setClauses, fmt.Sprintf("reality_server_name = $%d", argNum))
		args = append(args, *req.RealityServerName)
		argNum++
	}
	if req.RealityPublicKey != nil {
		setClauses = append(setClauses, fmt.Sprintf("reality_public_key = $%d", argNum))
		args = append(args, *req.RealityPublicKey)
		argNum++
	}
	if req.RealityPort != nil {
		setClauses = append(setClauses, fmt.Sprintf("reality_port = $%d", argNum))
		args = append(args, *req.RealityPort)
		argNum++
	}

	if len(setClauses) == 0 {
		return r.GetByID(id)
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++

	args = append(args, id)

	query := fmt.Sprintf("UPDATE servers SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argNum)
	_, err := r.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}

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