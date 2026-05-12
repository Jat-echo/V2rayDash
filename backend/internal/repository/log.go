package repository

import (
	"database/sql"
	"time"

	"v2ray-dash/backend/internal/model"
)

type LogRepository struct {
	db *sql.DB
}

func NewLogRepository(db *sql.DB) *LogRepository {
	return &LogRepository{db: db}
}

func (r *LogRepository) Create(log *model.OperationLog) error {
	_, err := r.db.Exec(
		`INSERT INTO operation_logs (operator, action, target_type, target_id, detail, ip)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		log.Operator, log.Action, log.TargetType, log.TargetID, log.Detail, log.IP,
	)
	return err
}

func (r *LogRepository) List(filter *model.OperationLogFilter) ([]*model.OperationLog, error) {
	query := `SELECT id, operator, action, target_type, target_id, detail, ip, created_at
		 FROM operation_logs WHERE 1=1`

	args := []interface{}{}
	argIdx := 1

	if filter.StartTime != nil {
		query += ` AND created_at >= $` + string(rune('0'+argIdx))
		args = append(args, *filter.StartTime)
		argIdx++
	}
	// ... 其他过滤条件

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.OperationLog
	for rows.Next() {
		var l model.OperationLog
		if err := rows.Scan(&l.ID, &l.Operator, &l.Action, &l.TargetType, &l.TargetID, &l.Detail, &l.IP, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, nil
}

func (r *LogRepository) CreateNodeStatus(status *model.NodeStatus) error {
	_, err := r.db.Exec(
		`INSERT INTO node_status (server_id, cpu_percent, memory_percent, disk_percent, bandwidth_in, bandwidth_out, v2ray_status, reported_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		status.ServerID, status.CPUPercent, status.MemoryPercent, status.DiskPercent,
		status.BandwidthIn, status.BandwidthOut, status.V2rayStatus, time.Now(),
	)
	return err
}

func (r *LogRepository) GetLatestNodeStatus(serverID string) (*model.NodeStatus, error) {
	var s model.NodeStatus
	err := r.db.QueryRow(
		`SELECT id, server_id, cpu_percent, memory_percent, disk_percent, bandwidth_in, bandwidth_out, v2ray_status, reported_at
		 FROM node_status WHERE server_id = $1 ORDER BY reported_at DESC LIMIT 1`,
		serverID,
	).Scan(&s.ID, &s.ServerID, &s.CPUPercent, &s.MemoryPercent, &s.DiskPercent, &s.BandwidthIn, &s.BandwidthOut, &s.V2rayStatus, &s.ReportedAt)
	return &s, err
}