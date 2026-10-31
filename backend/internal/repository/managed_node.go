package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"v2ray-dash/backend/internal/model"
)

func (r *ServerRepository) GetNode(id string) (*model.ManagedNode, error) {
	n := &model.ManagedNode{}
	err := r.db.QueryRow(`SELECT server_id,role,layout,inbound_tag,short_id,egress_ip,state,last_error,config_hash,ssh_host_key,updated_at FROM managed_nodes WHERE server_id=$1`, id).Scan(&n.ServerID, &n.Role, &n.Layout, &n.InboundTag, &n.ShortID, &n.EgressIP, &n.State, &n.LastError, &n.ConfigHash, &n.SSHHostKey, &n.UpdatedAt)
	return n, err
}

func (r *ServerRepository) attachNode(s *model.Server) error {
	n, err := r.GetNode(s.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err == nil {
		s.Node = n
	}
	return err
}

func (r *ServerRepository) SaveNode(n *model.ManagedNode) error {
	_, err := r.db.Exec(`INSERT INTO managed_nodes (server_id,role,layout,inbound_tag,short_id,egress_ip,config_hash,ssh_host_key) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	ON CONFLICT(server_id) DO UPDATE SET role=EXCLUDED.role,layout=EXCLUDED.layout,inbound_tag=EXCLUDED.inbound_tag,short_id=EXCLUDED.short_id,egress_ip=EXCLUDED.egress_ip,config_hash=EXCLUDED.config_hash,ssh_host_key=EXCLUDED.ssh_host_key,state=CASE WHEN managed_nodes.state='pending' THEN 'pending' ELSE 'adopted' END,last_error='',updated_at=CURRENT_TIMESTAMP`, n.ServerID, n.Role, n.Layout, n.InboundTag, n.ShortID, n.EgressIP, n.ConfigHash, n.SSHHostKey)
	return err
}

func (r *ServerRepository) SetNodeState(id, state, detail, hash string) error {
	_, err := r.db.Exec(`UPDATE managed_nodes SET state=$2,last_error=$3,config_hash=CASE WHEN $4='' THEN config_hash ELSE $4 END,updated_at=CURRENT_TIMESTAMP WHERE server_id=$1`, id, state, detail, hash)
	return err
}

func (r *ServerRepository) BeginPublish(id string) (bool, error) {
	result, err := r.db.Exec(`UPDATE managed_nodes SET state='publishing',last_error='',updated_at=CURRENT_TIMESTAMP WHERE server_id=$1 AND state<>'publishing'`, id)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (r *ServerRepository) AccountReferenced(id string) (bool, error) {
	var used bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM relay_routes WHERE exit_account_id=$1 OR applied_exit_account_id=$1) OR EXISTS(SELECT 1 FROM relay_combinations WHERE exit_account_id=$1)`, id).Scan(&used)
	return used, err
}

func (r *ServerRepository) ListRoutes() ([]*model.RelayRoute, error) {
	rows, err := r.db.Query(`SELECT account_id,exit_account_id,COALESCE(applied_exit_account_id::text,''),revision,applied_revision FROM relay_routes ORDER BY account_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*model.RelayRoute, 0)
	for rows.Next() {
		v := &model.RelayRoute{}
		if err = rows.Scan(&v.AccountID, &v.ExitAccountID, &v.AppliedExitAccountID, &v.Revision, &v.AppliedRevision); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *ServerRepository) BindRoute(accountID, exitID string) error {
	// Keep the last applied target until the new configuration has actually passed validation.
	_, err := r.db.Exec(`INSERT INTO relay_routes (account_id,exit_account_id) VALUES ($1,$2) ON CONFLICT(account_id) DO UPDATE SET exit_account_id=EXCLUDED.exit_account_id,revision=relay_routes.revision+1 WHERE relay_routes.exit_account_id<>EXCLUDED.exit_account_id`, accountID, exitID)
	return err
}

func (r *ServerRepository) MarkRoutesApplied(routes []*model.RelayRoute) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, v := range routes {
		if _, err = tx.Exec(`UPDATE relay_routes SET applied_exit_account_id=$2,applied_revision=$3 WHERE account_id=$1 AND revision=$3`, v.AccountID, v.ExitAccountID, v.Revision); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *ServerRepository) SavePublishedAccounts(accounts []*model.Account, enabled map[string]bool) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, a := range accounts {
		if _, err = tx.Exec(`INSERT INTO managed_account_state (account_id,uuid,enabled) VALUES ($1,$2,$3) ON CONFLICT(account_id) DO UPDATE SET uuid=EXCLUDED.uuid,enabled=EXCLUDED.enabled`, a.ID, a.UUID, enabled[a.ID]); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *ServerRepository) CanExportAccount(a *model.Account) (bool, error) {
	if a.Transport {
		return false, nil
	}
	return r.AccountInstalled(a)
}

// AccountFlagSource follows the applied route so pending route changes do not relabel live nodes.
func (r *ServerRepository) AccountFlagSource(accountID string) (*model.Server, error) {
	var node model.Server
	err := r.db.QueryRow(`SELECT s.name,COALESCE(s.country_code,'') FROM accounts a
	 LEFT JOIN relay_routes rr ON rr.account_id=a.id
	 LEFT JOIN accounts upstream ON upstream.id=rr.applied_exit_account_id
	 JOIN servers s ON s.id=COALESCE(upstream.server_id,a.server_id)
	 WHERE a.id=$1`, accountID).Scan(&node.Name, &node.CountryCode)
	return &node, err
}

// FinalizePublish records only the configuration that passed remote verification.
func (r *ServerRepository) FinalizePublish(serverID, hash string, routes []*model.RelayRoute, accounts []*model.Account, enabled map[string]bool) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, v := range routes {
		if _, err = tx.Exec(`UPDATE relay_routes SET applied_exit_account_id=$2,applied_revision=$3 WHERE account_id=$1 AND revision=$3`, v.AccountID, v.ExitAccountID, v.Revision); err != nil {
			return err
		}
	}
	for _, a := range accounts {
		if _, err = tx.Exec(`INSERT INTO managed_account_state (account_id,uuid,enabled) VALUES ($1,$2,$3) ON CONFLICT(account_id) DO UPDATE SET uuid=EXCLUDED.uuid,enabled=EXCLUDED.enabled`, a.ID, a.UUID, enabled[a.ID]); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`UPDATE managed_nodes SET state='active',last_error='',config_hash=$2,updated_at=CURRENT_TIMESTAMP WHERE server_id=$1`, serverID, hash); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ServerRepository) AccountInstalled(a *model.Account) (bool, error) {
	var ready bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM accounts a LEFT JOIN managed_nodes n ON n.server_id=a.server_id LEFT JOIN managed_account_state p ON p.account_id=a.id LEFT JOIN relay_routes rr ON rr.account_id=a.id WHERE a.id=$1 AND a.enabled AND (n.server_id IS NULL OR (p.uuid=a.uuid AND p.enabled AND (n.role='exit' OR rr.applied_exit_account_id IS NOT NULL))))`, a.ID).Scan(&ready)
	return ready, err
}

func (r *AccountRepository) MarkTransport(id string) error {
	_, err := r.db.Exec(`UPDATE accounts SET transport=TRUE,stats_key='vd-'||uuid WHERE id=$1`, id)
	return err
}

func (r *ServerRepository) AgentTokenHash(id string) (string, error) {
	var value string
	err := r.db.QueryRow(`SELECT agent_token_hash FROM managed_nodes WHERE server_id=$1`, id).Scan(&value)
	return value, err
}

func (r *ServerRepository) SetAgentTokenHash(id, value string) error {
	_, err := r.db.Exec(`UPDATE managed_nodes SET agent_token_hash=$2 WHERE server_id=$1`, id, value)
	return err
}

func (r *AccountRepository) GetByUUID(value string) (*model.Account, error) {
	var id string
	if err := r.db.QueryRow(`SELECT id FROM accounts WHERE uuid=$1`, value).Scan(&id); err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func (r *AccountRepository) StatsKey(id string) (string, error) {
	var key string
	err := r.db.QueryRow(`SELECT COALESCE(NULLIF(stats_key,''),email) FROM accounts WHERE id=$1`, id).Scan(&key)
	return key, err
}

func (r *AccountRepository) SetStatsKey(id, key string) error {
	_, err := r.db.Exec(`UPDATE accounts SET stats_key=$2 WHERE id=$1`, id, key)
	return err
}

func (r *AccountRepository) EffectiveEnabled(id string) (bool, error) {
	var enabled bool
	err := r.db.QueryRow(`SELECT a.enabled AND (a.traffic_limit=0 OR a.traffic_used<a.traffic_limit) AND
	(NOT EXISTS(SELECT 1 FROM subscription_accounts WHERE account_id=a.id) OR EXISTS(
	 SELECT 1 FROM subscription_accounts sa JOIN subscriptions s ON s.id=sa.subscription_id
	 WHERE sa.account_id=a.id AND s.enable AND (s.traffic_limit=0 OR s.traffic_used<s.traffic_limit))) FROM accounts a WHERE a.id=$1`, id).Scan(&enabled)
	return enabled, err
}

func (r *ServerRepository) ValidateRoute(accountID, exitID string) error {
	var valid bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM accounts a JOIN managed_nodes n ON n.server_id=a.server_id JOIN accounts e ON e.id=$2 JOIN managed_nodes en ON en.server_id=e.server_id WHERE a.id=$1 AND n.role='relay' AND en.role='exit' AND e.enabled AND a.server_id<>e.server_id AND (a.combination_id IS NULL OR EXISTS(SELECT 1 FROM relay_combinations c WHERE c.id=a.combination_id AND c.relay_server_id=a.server_id AND c.exit_server_id=e.server_id AND c.exit_account_id=e.id)))`, accountID, exitID).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("请选择已接管出口节点中的启用账号；不支持中转到中转")
	}
	return nil
}

func (r *ServerRepository) CombinationExitServers(relayID string) ([]string, error) {
	rows, err := r.db.Query(`SELECT DISTINCT exit_server_id FROM relay_combinations WHERE relay_server_id=$1 ORDER BY exit_server_id`, relayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
