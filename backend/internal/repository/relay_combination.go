package repository

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"net"
	"net/url"
	"strconv"
	"strings"
	"v2ray-dash/backend/internal/model"
)

type CombinationRepository struct{ db *sql.DB }

func NewCombinationRepository(db *sql.DB) *CombinationRepository { return &CombinationRepository{db} }

func (r *CombinationRepository) List() ([]*model.RelayCombination, error) {
	rows, err := r.db.Query(`SELECT c.id,c.name,c.relay_server_id,c.exit_server_id,COALESCE(c.exit_account_id::text,''),
	 EXISTS(SELECT 1 FROM accounts a JOIN managed_account_state p ON p.account_id=a.id WHERE a.id=c.exit_account_id AND a.enabled AND p.enabled AND p.uuid=a.uuid),r.name,e.name,
	 (SELECT COUNT(*) FROM accounts a WHERE a.combination_id=c.id),
	 (SELECT COUNT(DISTINCT sa.subscription_id) FROM accounts a JOIN subscription_accounts sa ON sa.account_id=a.id WHERE a.combination_id=c.id),c.created_at
	 FROM relay_combinations c JOIN servers r ON r.id=c.relay_server_id JOIN servers e ON e.id=c.exit_server_id ORDER BY c.created_at,c.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*model.RelayCombination{}
	for rows.Next() {
		v := &model.RelayCombination{}
		if err = rows.Scan(&v.ID, &v.Name, &v.RelayServerID, &v.ExitServerID, &v.ExitAccountID, &v.ConnectionReady, &v.RelayServerName, &v.ExitServerName, &v.AccountCount, &v.SubscriptionCount, &v.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *CombinationRepository) Save(id string, req *model.CombinationRequest) (string, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len([]rune(req.Name)) > 100 {
		return "", fmt.Errorf("填写链路名称（最多 100 字）")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var relay, exit, upstream string
	if id != "" {
		if err = tx.QueryRow(`SELECT relay_server_id,exit_server_id,exit_account_id FROM relay_combinations WHERE id=$1 FOR UPDATE`, id).Scan(&relay, &exit, &upstream); err != nil {
			return "", err
		}
		if relay == req.RelayServerID && exit == req.ExitServerID {
			if _, err = tx.Exec(`UPDATE relay_combinations SET name=$2 WHERE id=$1`, id, req.Name); err != nil {
				return "", fmt.Errorf("链路名称已存在")
			}
			return id, tx.Commit()
		}
	}
	for _, pair := range []struct{ id, role string }{{req.RelayServerID, "relay"}, {req.ExitServerID, "exit"}} {
		var role string
		if err = tx.QueryRow(`SELECT role FROM servers WHERE id=$1 FOR SHARE`, pair.id).Scan(&role); err != nil || role != pair.role {
			return "", fmt.Errorf("请选择中转节点和落地节点，不能创建中转到中转的链路")
		}
		var state string
		err = tx.QueryRow(`SELECT state FROM managed_nodes WHERE server_id=$1 FOR UPDATE`, pair.id).Scan(&state)
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
		if state == "publishing" {
			return "", fmt.Errorf("节点正在发布，请稍后修改链路")
		}
		if pair.role == "exit" && err == sql.ErrNoRows {
			return "", fmt.Errorf("请先接管落地服务，读取真实连接参数")
		}
	}
	if id == "" {
		id = uuid.NewString()
		upstream, e := createCombinationConnection(tx, id, req.ExitServerID)
		if e != nil {
			return "", e
		}
		_, err = tx.Exec(`INSERT INTO relay_combinations(id,name,relay_server_id,exit_server_id,exit_account_id) VALUES($1,$2,$3,$4,$5)`, id, req.Name, req.RelayServerID, req.ExitServerID, upstream)
	} else {
		var count int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM accounts WHERE combination_id=$1`, id).Scan(&count); err != nil {
			return "", err
		}
		if count > 0 && (relay != req.RelayServerID || exit != req.ExitServerID) {
			return "", fmt.Errorf("链路已有账号，不能更换节点；请新建链路再给订阅添加节点")
		}
		oldUpstream := upstream
		if exit != req.ExitServerID {
			var state string
			if err = tx.QueryRow(`SELECT state FROM managed_nodes WHERE server_id=$1 FOR UPDATE`, exit).Scan(&state); err != nil {
				return "", err
			}
			if state == "publishing" {
				return "", fmt.Errorf("原落地正在发布，请稍后更换")
			}
			upstream, err = createCombinationConnection(tx, id, req.ExitServerID)
			if err != nil {
				return "", err
			}
		}
		_, err = tx.Exec(`UPDATE relay_combinations SET name=$2,relay_server_id=$3,exit_server_id=$4,exit_account_id=$5 WHERE id=$1`, id, req.Name, req.RelayServerID, req.ExitServerID, upstream)
		if err == nil && oldUpstream != upstream {
			if _, err = tx.Exec(`DELETE FROM accounts WHERE id=$1`, oldUpstream); err != nil {
				return "", err
			}
			if _, err = tx.Exec(`UPDATE managed_nodes SET state='pending',last_error='',updated_at=CURRENT_TIMESTAMP WHERE server_id=$1`, exit); err != nil {
				return "", err
			}
		}
	}
	if err != nil {
		return "", fmt.Errorf("链路名称或中转与落地的搭配已存在")
	}
	return id, tx.Commit()
}

func createCombinationConnection(tx *sql.Tx, combinationID, exitID string) (string, error) {
	var id string
	credential := uuid.NewString()
	err := tx.QueryRow(`INSERT INTO accounts(server_id,uuid,email,protocols,stats_key,transport) VALUES($1,$2,$3,ARRAY['vless_reality_vision'],$4,TRUE) RETURNING id`, exitID, credential, "combination-"+combinationID, "vd-"+credential).Scan(&id)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(`UPDATE managed_nodes SET state='pending',last_error='',updated_at=CURRENT_TIMESTAMP WHERE server_id=$1`, exitID)
	return id, err
}

func (r *CombinationRepository) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var upstream, exit string
	if err = tx.QueryRow(`SELECT exit_account_id,exit_server_id FROM relay_combinations WHERE id=$1 FOR UPDATE`, id).Scan(&upstream, &exit); err != nil {
		return err
	}
	var state string
	if err = tx.QueryRow(`SELECT state FROM managed_nodes WHERE server_id=$1 FOR UPDATE`, exit).Scan(&state); err != nil {
		return err
	}
	if state == "publishing" {
		return fmt.Errorf("落地正在发布，请稍后删除")
	}
	if _, err = tx.Exec(`DELETE FROM relay_combinations WHERE id=$1`, id); err != nil {
		return fmt.Errorf("链路仍有账号，请先移除关联节点并删除账号")
	}
	if _, err = tx.Exec(`DELETE FROM accounts WHERE id=$1`, upstream); err != nil {
		return fmt.Errorf("连接仍被引用，请先清理关联账号")
	}
	if _, err = tx.Exec(`UPDATE managed_nodes SET state='pending',last_error='',updated_at=CURRENT_TIMESTAMP WHERE server_id=$1`, exit); err != nil {
		return err
	}
	return tx.Commit()
}

// Connection is admin-only. Customer subscriptions never include this transport credential.
func (r *CombinationRepository) Connection(id string) (*model.RelayConnection, error) {
	v := &model.RelayConnection{Protocol: "vless", Security: "reality"}
	var name string
	err := r.db.QueryRow(`SELECT c.name,s.ip,s.reality_port,a.uuid,s.reality_server_name,s.reality_public_key,n.short_id,
	 EXISTS(SELECT 1 FROM managed_account_state p WHERE p.account_id=a.id AND p.enabled AND p.uuid=a.uuid AND a.enabled)
	 FROM relay_combinations c JOIN accounts a ON a.id=c.exit_account_id JOIN servers s ON s.id=c.exit_server_id JOIN managed_nodes n ON n.server_id=s.id WHERE c.id=$1`, id).Scan(&name, &v.Address, &v.Port, &v.UUID, &v.ServerName, &v.PublicKey, &v.ShortID, &v.Ready)
	if err != nil {
		return nil, err
	}
	params := url.Values{"encryption": {"none"}, "security": {"reality"}, "type": {"tcp"}, "flow": {"xtls-rprx-vision"}, "sni": {v.ServerName}, "fp": {"chrome"}, "pbk": {v.PublicKey}, "sid": {v.ShortID}}
	u := url.URL{Scheme: "vless", User: url.User(v.UUID), Host: net.JoinHostPort(v.Address, strconv.Itoa(v.Port)), RawQuery: params.Encode(), Fragment: name + " · 上游连接"}
	v.URI = u.String()
	return v, nil
}
