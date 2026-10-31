package service

import (
	"context"
	"database/sql"
	"log"
	"time"

	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

// RunNodeAutoSync uses persisted pending states; failed jobs wait for a new change or manual retry.
// ponytail: one controller and the existing node-operation lock suit this private panel.
func RunNodeAutoSync(ctx context.Context, db *sql.DB, quiet, poll time.Duration) {
	servers := repository.NewServerRepository(db)
	nodes := NewNodeService(repository.NewAccountRepository(db), servers)
	logs := repository.NewLogRepository(db)
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		ids, err := servers.PendingNodes(quiet)
		if err != nil {
			log.Printf("auto sync: load pending nodes: %v", err)
			continue
		}
		for _, id := range ids {
			if ctx.Err() != nil {
				return
			}
			started, checks, err := nodes.publishAutomatic(id, quiet)
			if err != nil && !started {
				log.Printf("auto sync: claim node %s: %v", id, err)
			}
			if !started {
				continue
			}
			detail := map[string]any{"checks": checks}
			if err != nil {
				detail["error"] = err.Error()
				log.Printf("auto sync node %s failed: %v", id, err)
			}
			if e := logs.Create(&model.OperationLog{Operator: "system", Action: "auto_sync_node", TargetType: "server", TargetID: id, Detail: detail}); e != nil {
				log.Printf("auto sync: save result: %v", e)
			}
		}
	}
}

func (s *NodeService) publishAutomatic(id string, quiet time.Duration) (bool, []map[string]any, error) {
	if !nodeOperationMu.TryLock() {
		return false, nil, nil
	}
	defer nodeOperationMu.Unlock()
	started, err := s.servers.BeginAutoPublish(id, quiet)
	if err != nil || !started {
		return false, nil, err
	}
	checks, err := s.publishLocked(id)
	return true, checks, err
}
