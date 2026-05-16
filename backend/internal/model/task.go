package model

import (
    "time"
)

type InstallTask struct {
    ID          string     `json:"id"`
    ServerID    string     `json:"server_id"`
    Status      string     `json:"status"` // pending, running, success, failed
    Output      string     `json:"output"`
    Error       string     `json:"error"`
    StartedAt   time.Time  `json:"started_at"`
    CompletedAt *time.Time `json:"completed_at"`
}