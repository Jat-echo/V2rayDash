package model

import (
	"time"
)

type Template struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      TemplateConfig `json:"config"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type TemplateConfig struct {
	Core           string   `json:"core"`            // "sing-box" or "xray-core"
	Port           int      `json:"port"`             // default 443
	UUID           string   `json:"uuid"`             // empty = auto
	ServerName     string   `json:"server_name"`      // Reality target domain
	Protocols      []string `json:"protocols"`        // e.g. ["vless_reality_vision", "vless_tcp_vision"]
	AgentEnabled   bool     `json:"agent_enabled"`    // default false
	ReportInterval int      `json:"report_interval"`  // default 30
}