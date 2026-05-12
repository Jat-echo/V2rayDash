package model

import (
	"time"
)

type NodeStatus struct {
	ID           string    `json:"id"`
	ServerID     string    `json:"server_id"`
	CPUPercent   float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	DiskPercent  float64   `json:"disk_percent"`
	BandwidthIn  int64     `json:"bandwidth_in"`
	BandwidthOut int64     `json:"bandwidth_out"`
	V2rayStatus  string    `json:"v2ray_status"`
	ReportedAt   time.Time `json:"reported_at"`
}

type HeartbeatRequest struct {
	ServerID    string  `json:"server_id" binding:"required"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemPercent  float64 `json:"mem_percent"`
	DiskPercent float64 `json:"disk_percent"`
	BandwidthIn int64   `json:"bandwidth_in"`
	BandwidthOut int64  `json:"bandwidth_out"`
	V2rayStatus string  `json:"v2ray_status"`
}