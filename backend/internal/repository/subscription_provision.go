package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"v2ray-dash/backend/internal/model"
)

// CreateWithTargets commits the subscription and its customer accounts together.
// A failed target leaves no partial subscription behind.
func (r *SubscriptionRepository) CreateWithTargets(req *model.CreateSubscriptionRequest) (string, []string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback()
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 100 || req.TrafficLimit < 0 {
		return "", nil, fmt.Errorf("订阅名称或流量限制无效")
	}
	if _, err = tx.Exec(`SELECT pg_advisory_xact_lock(hashtext($1))`, name); err != nil {
		return "", nil, err
	}
	var duplicate bool
	if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM subscriptions WHERE name=$1)`, name).Scan(&duplicate); err != nil {
		return "", nil, err
	}
	if duplicate {
		return "", nil, fmt.Errorf("已存在同名订阅，请改用添加到已有订阅")
	}
	var id string
	if err = tx.QueryRow(`INSERT INTO subscriptions(name,remark,uuid,traffic_limit) VALUES($1,$2,$3,$4) RETURNING id`, name, req.Remark, uuid.NewString(), req.TrafficLimit).Scan(&id); err != nil {
		return "", nil, err
	}
	servers := []string{}
	for _, m := range req.AccountMappings {
		_, affected, e := assignSubscriptionTarget(tx, id, m)
		if e != nil {
			return "", nil, e
		}
		servers = append(servers, affected...)
	}
	return id, servers, tx.Commit()
}

func (r *SubscriptionRepository) AddTarget(id string, m model.AccountMapping) (bool, []string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, nil, err
	}
	defer tx.Rollback()
	added, servers, err := assignSubscriptionTarget(tx, id, m)
	if err != nil {
		return false, nil, err
	}
	return added, servers, tx.Commit()
}

// RemoveTarget removes only this path. Accounts shared by other subscriptions and
// the combination's upstream transport remain intact.
func (r *SubscriptionRepository) RemoveTarget(id string, m model.AccountMapping) (bool, []string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, nil, err
	}
	defer tx.Rollback()
	var subID string
	if err = tx.QueryRow(`SELECT id FROM subscriptions WHERE id=$1 FOR UPDATE`, id).Scan(&subID); err != nil {
		return false, nil, fmt.Errorf("订阅不存在")
	}
	serverID := m.ServerID
	if m.CombinationID != "" {
		if err = tx.QueryRow(`SELECT relay_server_id FROM relay_combinations WHERE id=$1 FOR SHARE`, m.CombinationID).Scan(&serverID); err != nil {
			return false, nil, fmt.Errorf("代理链路不存在")
		}
	} else {
		var role string
		if err = tx.QueryRow(`SELECT role FROM servers WHERE id=$1 FOR SHARE`, serverID).Scan(&role); err != nil || role != "exit" {
			return false, nil, fmt.Errorf("请选择直连节点或代理链路")
		}
	}
	var state string
	err = tx.QueryRow(`SELECT state FROM managed_nodes WHERE server_id=$1 FOR UPDATE`, serverID).Scan(&state)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, nil, err
	}
	if state == "publishing" {
		return false, nil, fmt.Errorf("节点正在同步，请稍后解除关联")
	}
	rows, err := tx.Query(`SELECT a.id FROM accounts a JOIN subscription_accounts sa ON sa.account_id=a.id WHERE sa.subscription_id=$1 AND NOT a.transport AND (($2<>'' AND a.combination_id::text=$2) OR ($2='' AND a.server_id=$3 AND a.combination_id IS NULL)) ORDER BY a.id FOR UPDATE OF a`, id, m.CombinationID, serverID)
	if err != nil {
		return false, nil, err
	}
	ids := []string{}
	for rows.Next() {
		var accountID string
		if err = rows.Scan(&accountID); err != nil {
			rows.Close()
			return false, nil, err
		}
		ids = append(ids, accountID)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return false, nil, err
	}
	if len(ids) == 0 {
		return false, nil, tx.Commit()
	}
	for _, accountID := range ids {
		if _, err = tx.Exec(`DELETE FROM subscription_accounts WHERE subscription_id=$1 AND account_id=$2`, id, accountID); err != nil {
			return false, nil, err
		}
		if _, err = tx.Exec(`DELETE FROM accounts a WHERE a.id=$1 AND NOT EXISTS(SELECT 1 FROM subscription_accounts WHERE account_id=a.id)`, accountID); err != nil {
			return false, nil, fmt.Errorf("账号仍被上游连接引用，无法删除；关联未变更")
		}
	}
	if _, err = tx.Exec(`UPDATE managed_nodes SET state='pending',last_error='',updated_at=CURRENT_TIMESTAMP WHERE server_id=$1`, serverID); err != nil {
		return false, nil, err
	}
	return true, []string{serverID}, tx.Commit()
}

func (r *SubscriptionRepository) ReplaceTargets(id string, mappings []model.AccountMapping) ([]string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var exists string
	if err = tx.QueryRow(`SELECT id FROM subscriptions WHERE id=$1 FOR UPDATE`, id).Scan(&exists); err != nil {
		return nil, err
	}
	previous, err := subscriptionNodeIDs(tx, id)
	if err != nil {
		return nil, err
	}
	if err = lockManagedNodes(tx, previous); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`DELETE FROM subscription_accounts WHERE subscription_id=$1`, id); err != nil {
		return nil, err
	}
	affected := append([]string{}, previous...)
	for _, m := range mappings {
		_, servers, e := assignSubscriptionTarget(tx, id, m)
		if e != nil {
			return nil, e
		}
		affected = append(affected, servers...)
	}
	if err = markNodesPending(tx, previous); err != nil {
		return nil, err
	}
	return affected, tx.Commit()
}

func assignSubscriptionTarget(tx *sql.Tx, subID string, m model.AccountMapping) (bool, []string, error) {
	var name string
	if err := tx.QueryRow(`SELECT name FROM subscriptions WHERE id=$1 FOR UPDATE`, subID).Scan(&name); err != nil {
		return false, nil, fmt.Errorf("订阅不存在")
	}
	serverID, exitID, upstreamID := m.ServerID, "", ""
	if m.CombinationID != "" {
		if m.ServerID != "" || m.AccountID != "" {
			return false, nil, fmt.Errorf("链路节点不能同时指定直连节点或已有账号")
		}
		if err := tx.QueryRow(`SELECT relay_server_id,exit_server_id,exit_account_id FROM relay_combinations WHERE id=$1 FOR SHARE`, m.CombinationID).Scan(&serverID, &exitID, &upstreamID); err != nil {
			return false, nil, fmt.Errorf("代理链路不存在，请先手动创建")
		}
	} else if serverID == "" {
		return false, nil, fmt.Errorf("请选择直连落地节点或代理链路")
	}
	affected := []string{serverID}
	if exitID != "" {
		affected = append(affected, exitID)
	}
	for _, sid := range affected {
		var role string
		if err := tx.QueryRow(`SELECT role FROM servers WHERE id=$1 FOR SHARE`, sid).Scan(&role); err != nil {
			return false, nil, fmt.Errorf("节点不存在")
		}
		want := "exit"
		if exitID != "" && sid == serverID {
			want = "relay"
		}
		if role != want {
			return false, nil, fmt.Errorf("直连只能选择落地节点，中转请使用已创建的链路")
		}
		var state, nodeRole string
		err := tx.QueryRow(`SELECT state,role FROM managed_nodes WHERE server_id=$1 FOR UPDATE`, sid).Scan(&state, &nodeRole)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, nil, err
		}
		if state == "publishing" {
			return false, nil, fmt.Errorf("节点正在发布，请稍后添加")
		}
		if exitID != "" && (errors.Is(err, sql.ErrNoRows) || nodeRole != want) {
			return false, nil, fmt.Errorf("请先在节点管理中接管链路的中转与落地服务")
		}
	}
	accountID := m.AccountID
	if accountID != "" {
		var valid bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND server_id=$2 AND NOT transport AND combination_id IS NULL)`, accountID, serverID).Scan(&valid); err != nil {
			return false, nil, err
		}
		if !valid {
			return false, nil, fmt.Errorf("账号不属于所选直连节点，或是内部传输账号")
		}

	}
	var linked bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM subscription_accounts sa JOIN accounts a ON a.id=sa.account_id WHERE sa.subscription_id=$1 AND NOT a.transport AND (($2<>'' AND a.combination_id::text=$2) OR ($2='' AND a.server_id=$3 AND a.combination_id IS NULL)))`, subID, m.CombinationID, serverID).Scan(&linked); err != nil {
		return false, nil, err
	}
	if linked {
		if accountID != "" {
			var same bool
			if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM subscription_accounts WHERE subscription_id=$1 AND account_id=$2)`, subID, accountID).Scan(&same); err != nil {
				return false, nil, err
			}
			if !same {
				return false, nil, fmt.Errorf("订阅已选择此直连节点的另一个账号")
			}
		}
		return false, nil, nil
	}
	if accountID == "" {

		if !m.AutoCreate && m.CombinationID == "" {
			return false, nil, fmt.Errorf("请选择已有账号或自动创建账号")
		}
		// Reuse the same customer account if this target is removed and added again.
		{
			err := tx.QueryRow(`SELECT id FROM accounts WHERE server_id=$1 AND email=$2 AND combination_id IS NOT DISTINCT FROM NULLIF($3,'')::uuid AND NOT transport ORDER BY created_at LIMIT 1`, serverID, name, m.CombinationID).Scan(&accountID)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return false, nil, err
			}
		}
		if accountID == "" {
			accountUUID := uuid.NewString()
			statsKey := ""
			if exitID != "" {
				statsKey = "vd-" + accountUUID
			}
			if err := tx.QueryRow(`INSERT INTO accounts(server_id,uuid,email,protocols,stats_key,combination_id) VALUES($1,$2,$3,ARRAY['vless_reality_vision'],$4,NULLIF($5,'')::uuid) RETURNING id`, serverID, accountUUID, name, statsKey, m.CombinationID).Scan(&accountID); err != nil {
				return false, nil, err
			}
			if exitID != "" {
				if _, err := tx.Exec(`INSERT INTO relay_routes(account_id,exit_account_id) VALUES($1,$2)`, accountID, upstreamID); err != nil {
					return false, nil, err
				}
			}
		}
	}
	if _, err := tx.Exec(`INSERT INTO subscription_accounts(subscription_id,account_id,sort_order) SELECT $1,$2,COALESCE(MAX(sort_order)+1,0) FROM subscription_accounts WHERE subscription_id=$1`, subID, accountID); err != nil {
		return false, nil, err
	}
	if _, err := tx.Exec(`UPDATE subscriptions SET server_id=COALESCE(server_id,$2::uuid),updated_at=CURRENT_TIMESTAMP WHERE id=$1`, subID, serverID); err != nil {
		return false, nil, err
	}
	for _, sid := range affected {
		if sid == exitID {
			var installed bool
			if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM accounts a JOIN managed_account_state p ON p.account_id=a.id WHERE a.id=$1 AND a.enabled AND p.enabled AND p.uuid=a.uuid)`, upstreamID).Scan(&installed); err != nil {
				return false, nil, err
			}
			if installed {
				continue
			}
		}
		if _, err := tx.Exec(`UPDATE managed_nodes SET state='pending',last_error='',updated_at=CURRENT_TIMESTAMP WHERE server_id=$1`, sid); err != nil {
			return false, nil, err
		}
	}
	return true, affected, nil
}
