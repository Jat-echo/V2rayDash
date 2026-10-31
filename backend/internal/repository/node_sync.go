package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// Mutations lock the same rows as the publisher claim before changing accounts.
func lockManagedNodes(tx *sql.Tx, ids []string) error {
	rows, err := tx.Query(`SELECT state FROM managed_nodes WHERE server_id=ANY($1::uuid[]) ORDER BY server_id FOR UPDATE`, pq.Array(ids))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			return err
		}
		if state == "publishing" {
			return fmt.Errorf("节点正在同步，请稍后修改")
		}
	}
	return rows.Err()
}

func markNodesPending(tx *sql.Tx, ids []string) error {
	_, err := tx.Exec(`UPDATE managed_nodes SET state='pending',last_error='',updated_at=CURRENT_TIMESTAMP WHERE server_id=ANY($1::uuid[]) AND state<>'publishing'`, pq.Array(ids))
	return err
}

func subscriptionNodeIDs(tx *sql.Tx, id string) ([]string, error) {
	rows, err := tx.Query(`SELECT DISTINCT a.server_id FROM accounts a JOIN subscription_accounts sa ON sa.account_id=a.id WHERE sa.subscription_id=$1 ORDER BY a.server_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var serverID string
		if err = rows.Scan(&serverID); err != nil {
			return nil, err
		}
		ids = append(ids, serverID)
	}
	return ids, rows.Err()
}

func (r *ServerRepository) MarkNodePending(id string) error {
	_, err := r.db.Exec(`UPDATE managed_nodes SET state='pending',last_error='',updated_at=CURRENT_TIMESTAMP WHERE server_id=$1 AND state<>'publishing'`, id)
	return err
}

func (r *ServerRepository) PendingNodes(quiet time.Duration) ([]string, error) {
	rows, err := r.db.Query(`SELECT server_id FROM managed_nodes WHERE state='pending' AND updated_at<=CURRENT_TIMESTAMP-($1 * interval '1 second') ORDER BY CASE role WHEN 'exit' THEN 0 ELSE 1 END,updated_at,server_id`, quiet.Seconds())
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
	return ids, rows.Err()
}

func (r *ServerRepository) BeginAutoPublish(id string, quiet time.Duration) (bool, error) {
	result, err := r.db.Exec(`UPDATE managed_nodes n SET state='publishing',last_error='',updated_at=CURRENT_TIMESTAMP
	 WHERE n.server_id=$1 AND n.state='pending' AND n.updated_at<=CURRENT_TIMESTAMP-($2 * interval '1 second')
	 AND NOT EXISTS(SELECT 1 FROM managed_nodes e WHERE e.server_id IN (
	  SELECT c.exit_server_id FROM relay_combinations c WHERE c.relay_server_id=n.server_id
	  UNION SELECT a.server_id FROM relay_routes rr JOIN accounts r ON r.id=rr.account_id JOIN accounts a ON a.id=rr.exit_account_id WHERE r.server_id=n.server_id
	 ) AND (e.state='publishing' OR (e.state='pending' AND e.updated_at>CURRENT_TIMESTAMP-($2 * interval '1 second'))))`, id, quiet.Seconds())
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}
