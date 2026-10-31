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
	if req.Role == "" {
		req.Role = "exit"
	}
	if req.Role != "exit" && req.Role != "relay" {
		return nil, fmt.Errorf("节点角色无效")
	}
	if err := r.validateSSHJump("", req.SSHJumpServerID); err != nil {
		return nil, err
	}
	sshKeyType := req.SSHKeyType
	if sshKeyType == "" {
		sshKeyType = "key" // default to key-based auth
	}
	result := r.db.QueryRow(
		`INSERT INTO servers (name, ip, country_code, ssh_port, ssh_user, ssh_key_type, ssh_key, ssh_password, tags, reality_enabled, reality_server_name, reality_public_key, reality_port, role, ssh_jump_server_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NULLIF($15,'')::uuid)
		 RETURNING id, name, ip, country_code, ssh_port, ssh_user, ssh_key_type, tags, status, reality_enabled, reality_server_name, reality_public_key, reality_port, created_at, updated_at, role, COALESCE(ssh_jump_server_id::text,'')`,
		req.Name, req.IP, req.CountryCode, req.SSHPort, req.SSHUser, sshKeyType, req.SSHKey, req.SSHPassword, tagsJSON, req.RealityEnabled, req.RealityServerName, req.RealityPublicKey, req.RealityPort, req.Role, req.SSHJumpServerID,
	)

	var s model.Server
	var tagsBytes []byte
	err := result.Scan(&s.ID, &s.Name, &s.IP, &s.CountryCode, &s.SSHPort, &s.SSHUser, &s.SSHKeyType, &tagsBytes, &s.Status, &s.RealityEnabled, &s.RealityServerName, &s.RealityPublicKey, &s.RealityPort, &s.CreatedAt, &s.UpdatedAt, &s.Role, &s.SSHJumpServerID)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tagsBytes, &s.Tags)
	if err := r.attachNode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ServerRepository) GetByID(id string) (*model.Server, error) {
	var s model.Server
	var tagsBytes []byte
	err := r.db.QueryRow(
		`SELECT id, name, ip, country_code, ssh_port, ssh_user, ssh_key_type, tags, status, reality_enabled, reality_server_name, reality_public_key, reality_port, created_at, updated_at, role, COALESCE(ssh_jump_server_id::text,'')
		 FROM servers WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.Name, &s.IP, &s.CountryCode, &s.SSHPort, &s.SSHUser, &s.SSHKeyType, &tagsBytes, &s.Status, &s.RealityEnabled, &s.RealityServerName, &s.RealityPublicKey, &s.RealityPort, &s.CreatedAt, &s.UpdatedAt, &s.Role, &s.SSHJumpServerID)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tagsBytes, &s.Tags)
	if err := r.attachNode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// GetByIDForInstall 获取服务器完整信息（包括敏感字段），仅供内部使用
func (r *ServerRepository) GetByIDForInstall(id string) (*model.Server, error) {
	var s model.Server
	var tagsBytes []byte
	err := r.db.QueryRow(
		`SELECT id, name, ip, country_code, ssh_port, ssh_user, ssh_key_type, COALESCE(ssh_key,''), COALESCE(ssh_password,''), tags, status, reality_enabled, reality_server_name, reality_public_key, reality_port, created_at, updated_at, role, COALESCE(ssh_jump_server_id::text,'')
		 FROM servers WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.Name, &s.IP, &s.CountryCode, &s.SSHPort, &s.SSHUser, &s.SSHKeyType, &s.SSHKey, &s.SSHPassword, &tagsBytes, &s.Status, &s.RealityEnabled, &s.RealityServerName, &s.RealityPublicKey, &s.RealityPort, &s.CreatedAt, &s.UpdatedAt, &s.Role, &s.SSHJumpServerID)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tagsBytes, &s.Tags)
	if err := r.attachNode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ServerRepository) List() ([]*model.Server, error) {
	rows, err := r.db.Query(
		`SELECT id, name, ip, country_code, ssh_port, ssh_user, ssh_key_type, tags, status, reality_enabled, reality_server_name, reality_public_key, reality_port, created_at, updated_at, role, COALESCE(ssh_jump_server_id::text,'')
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
		if err := rows.Scan(&s.ID, &s.Name, &s.IP, &s.CountryCode, &s.SSHPort, &s.SSHUser, &s.SSHKeyType, &tagsBytes, &s.Status, &s.RealityEnabled, &s.RealityServerName, &s.RealityPublicKey, &s.RealityPort, &s.CreatedAt, &s.UpdatedAt, &s.Role, &s.SSHJumpServerID); err != nil {
			return nil, err
		}
		json.Unmarshal(tagsBytes, &s.Tags)
		servers = append(servers, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for _, server := range servers {
		if err := r.attachNode(server); err != nil {
			return nil, err
		}
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

	if req.Role != nil {
		if *req.Role != "exit" && *req.Role != "relay" {
			return nil, fmt.Errorf("节点角色无效")
		}
		var blocked bool
		err := r.db.QueryRow(`SELECT s.role<>$2 AND (EXISTS(SELECT 1 FROM servers WHERE ssh_jump_server_id=$1 AND $2<>'relay') OR EXISTS(SELECT 1 FROM relay_combinations WHERE (relay_server_id=$1 AND $2<>'relay') OR (exit_server_id=$1 AND $2<>'exit')) OR EXISTS(SELECT 1 FROM managed_nodes WHERE server_id=$1 AND role<>$2) OR ($2='relay' AND EXISTS(SELECT 1 FROM accounts WHERE server_id=$1 AND combination_id IS NULL AND NOT transport))) FROM servers s WHERE s.id=$1`, id, *req.Role).Scan(&blocked)
		if err != nil {
			return nil, err
		}
		if blocked {
			return nil, fmt.Errorf("节点已有链路、已接管服务或直连账号，不能变更角色")
		}
		setClauses = append(setClauses, fmt.Sprintf("role = $%d", argNum))
		args = append(args, *req.Role)
		argNum++
	}
	if req.SSHJumpServerID != nil {
		if err := r.validateSSHJump(id, *req.SSHJumpServerID); err != nil {
			return nil, err
		}
		var publishing bool
		if err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM servers s JOIN managed_nodes n ON n.server_id=s.id WHERE s.id=$1 AND n.state='publishing' AND COALESCE(s.ssh_jump_server_id::text,'')<>$2)`, id, *req.SSHJumpServerID).Scan(&publishing); err != nil {
			return nil, err
		}
		if publishing {
			return nil, fmt.Errorf("节点正在同步，请稍后修改 SSH 跳板")
		}
		setClauses = append(setClauses, fmt.Sprintf("ssh_jump_server_id = NULLIF($%d,'')::uuid", argNum))
		args = append(args, *req.SSHJumpServerID)
		argNum++
	}
	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *req.Name)
		argNum++
	}
	if req.IP != nil {
		setClauses = append(setClauses, fmt.Sprintf("ip = $%d", argNum))
		args = append(args, *req.IP)
		argNum++
	}
	if req.CountryCode != nil {
		setClauses = append(setClauses, fmt.Sprintf("country_code = $%d", argNum))
		args = append(args, *req.CountryCode)
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

func (r *ServerRepository) validateSSHJump(id, jumpID string) error {
	if jumpID == "" {
		return nil
	}
	if jumpID == id {
		return fmt.Errorf("SSH 跳板不能选择节点自身")
	}
	var role, parent, hostKey string
	if err := r.db.QueryRow(`SELECT s.role,COALESCE(s.ssh_jump_server_id::text,''),COALESCE(n.ssh_host_key,'') FROM servers s LEFT JOIN managed_nodes n ON n.server_id=s.id WHERE s.id=$1`, jumpID).Scan(&role, &parent, &hostKey); err != nil {
		return fmt.Errorf("SSH 跳板节点不存在")
	}
	if role != "relay" || parent != "" || hostKey == "" {
		return fmt.Errorf("请选择已读取配置且可直接连接的中转节点作为 SSH 跳板")
	}
	if id != "" {
		var used bool
		if err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM servers WHERE ssh_jump_server_id=$1)`, id).Scan(&used); err != nil {
			return err
		}
		if used {
			return fmt.Errorf("该节点已被用作 SSH 跳板，暂不支持多级跳板")
		}
	}
	return nil
}

func (r *ServerRepository) UpdateStatus(id, status string) error {
	_, err := r.db.Exec(
		`UPDATE servers SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now(), id,
	)
	return err
}
