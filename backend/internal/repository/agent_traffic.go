package repository

import (
	"fmt"
	"math"
	"v2ray-dash/backend/internal/model"
)

// RecordManagedTraffic applies cumulative counters and batch deduplication in one transaction.
func (r *ServerRepository) RecordManagedTraffic(req *model.HeartbeatRequest) (bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`INSERT INTO agent_traffic_batches (server_id,batch_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, req.ServerID, req.BatchID)
	if err != nil {
		return false, err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if inserted == 0 {
		return false, tx.Commit()
	}
	for _, stat := range req.UserTrafficStats {
		if stat.Upload < 0 || stat.Download < 0 || stat.Upload > math.MaxInt64-stat.Download {
			return false, fmt.Errorf("invalid traffic counters")
		}
		if req.TrafficEpoch == "" {
			return false, fmt.Errorf("missing counter epoch")
		}
		var accountID string
		rows, err := tx.Query(`SELECT id FROM accounts WHERE server_id=$1 AND COALESCE(NULLIF(stats_key,''),email)=$2 AND NOT transport AND NOT EXISTS(SELECT 1 FROM relay_routes rr WHERE rr.exit_account_id=accounts.id OR rr.applied_exit_account_id=accounts.id)`, req.ServerID, stat.Email)
		if err != nil {
			return false, err
		}
		if rows.Next() {
			err = rows.Scan(&accountID)
		}
		rows.Close()
		if err != nil {
			return false, err
		}
		if accountID == "" {
			continue
		}
		_, err = tx.Exec(`INSERT INTO agent_traffic_cursors (server_id,account_id,epoch) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, req.ServerID, accountID, req.TrafficEpoch)
		if err != nil {
			return false, err
		}
		var upload, download int64
		if err = tx.QueryRow(`SELECT upload,download FROM agent_traffic_cursors WHERE server_id=$1 AND account_id=$2 AND epoch=$3 FOR UPDATE`, req.ServerID, accountID, req.TrafficEpoch).Scan(&upload, &download); err != nil {
			return false, err
		}
		// A stale retry cannot move the cursor backwards.
		newUpload, newDownload := max(upload, stat.Upload), max(download, stat.Download)
		delta := (newUpload - upload) + (newDownload - download)
		if _, err = tx.Exec(`UPDATE agent_traffic_cursors SET upload=$4,download=$5 WHERE server_id=$1 AND account_id=$2 AND epoch=$3`, req.ServerID, accountID, req.TrafficEpoch, newUpload, newDownload); err != nil {
			return false, err
		}
		if delta == 0 {
			continue
		}
		if _, err = tx.Exec(`UPDATE accounts SET traffic_used=traffic_used+$2,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, accountID, delta); err != nil {
			return false, err
		}
		if _, err = tx.Exec(`INSERT INTO account_traffic_logs(account_id,traffic_bytes) SELECT id,traffic_used FROM accounts WHERE id=$1`, accountID); err != nil {
			return false, err
		}
	}
	if _, err = tx.Exec(`UPDATE subscriptions s SET traffic_used=(SELECT COALESCE(SUM(a.traffic_used),0) FROM accounts a JOIN subscription_accounts sa ON sa.account_id=a.id WHERE sa.subscription_id=s.id) WHERE EXISTS(SELECT 1 FROM subscription_accounts sa JOIN accounts a ON a.id=sa.account_id WHERE sa.subscription_id=s.id AND a.server_id=$1)`, req.ServerID); err != nil {
		return false, err
	}
	if _, err = tx.Exec(`INSERT INTO subscription_traffic_logs(subscription_id,traffic_bytes) SELECT s.id,s.traffic_used FROM subscriptions s WHERE EXISTS(SELECT 1 FROM subscription_accounts sa JOIN accounts a ON a.id=sa.account_id WHERE sa.subscription_id=s.id AND a.server_id=$1)`, req.ServerID); err != nil {
		return false, err
	}
	return true, tx.Commit()
}
