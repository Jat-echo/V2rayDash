package model

import (
	"time"
)

type OperationLog struct {
	ID         string    `json:"id"`
	Operator   string    `json:"operator"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Detail     map[string]any `json:"detail"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
}

type OperationLogFilter struct {
	StartTime *time.Time
	EndTime   *time.Time
	TargetType string
	Operator   string
}