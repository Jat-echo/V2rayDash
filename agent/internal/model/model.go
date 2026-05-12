package model

type NodeStatus struct {
	ServerID      string  `json:"server_id"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	BandwidthIn   int64   `json:"bandwidth_in"`
	BandwidthOut  int64   `json:"bandwidth_out"`
	V2rayStatus   string  `json:"v2ray_status"`
}