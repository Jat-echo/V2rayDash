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

type UserTrafficStat struct {
	Email    string `json:"email"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

type HeartbeatRequest struct {
	ServerID         string            `json:"server_id" binding:"required"`
	CPUPercent       float64           `json:"cpu_percent"`
	MemoryPercent    float64           `json:"memory_percent"`
	DiskPercent      float64           `json:"disk_percent"`
	BandwidthIn      int64             `json:"bandwidth_in"`
	BandwidthOut     int64             `json:"bandwidth_out"`
	V2rayStatus      string            `json:"v2ray_status"`
	UserTrafficStats []UserTrafficStat `json:"user_traffic_stats,omitempty"`
}

type MetricPoint struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

type BandwidthPoint struct {
	Time  time.Time `json:"time"`
	Value int64     `json:"value"`
}

type NodeStatusMetrics struct {
	CPU          []MetricPoint    `json:"cpu"`
	Memory       []MetricPoint    `json:"memory"`
	Disk         []MetricPoint    `json:"disk"`
	BandwidthIn  []BandwidthPoint `json:"bandwidth_in"`
	BandwidthOut []BandwidthPoint `json:"bandwidth_out"`
}

type NodeStatusCurrent struct {
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	DiskPercent   float64   `json:"disk_percent"`
	BandwidthIn   int64     `json:"bandwidth_in"`
	BandwidthOut  int64     `json:"bandwidth_out"`
	V2rayStatus   string    `json:"v2ray_status"`
	ReportedAt    time.Time `json:"reported_at"`
}

type NodeStatusResponse struct {
	ServerID        string             `json:"server_id"`
	Metrics         NodeStatusMetrics  `json:"metrics"`
	Current         *NodeStatusCurrent `json:"current"`
	V2rayRestarts   int                `json:"v2ray_restarts"` // 时间窗口内崩溃重启次数
}