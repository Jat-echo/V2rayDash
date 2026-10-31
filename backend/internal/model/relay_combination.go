package model

import "time"

// A reusable network path, created manually and independent of any customer account.
type RelayCombination struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	RelayServerID     string    `json:"relay_server_id"`
	ExitServerID      string    `json:"exit_server_id"`
	ExitAccountID     string    `json:"-"`
	ConnectionReady   bool      `json:"connection_ready"`
	RelayServerName   string    `json:"relay_server_name"`
	ExitServerName    string    `json:"exit_server_name"`
	AccountCount      int       `json:"account_count"`
	SubscriptionCount int       `json:"subscription_count"`
	CreatedAt         time.Time `json:"created_at"`
}

type RelayConnection struct {
	Protocol   string `json:"protocol"`
	Security   string `json:"security"`
	Address    string `json:"address"`
	Port       int    `json:"port"`
	UUID       string `json:"uuid"`
	ServerName string `json:"server_name"`
	PublicKey  string `json:"public_key"`
	ShortID    string `json:"short_id"`
	Ready      bool   `json:"ready"`
	URI        string `json:"uri"`
}

type CombinationRequest struct {
	Name          string `json:"name" binding:"required,max=100"`
	RelayServerID string `json:"relay_server_id" binding:"required,uuid"`
	ExitServerID  string `json:"exit_server_id" binding:"required,uuid"`
}

type BatchSubscriptionRequest struct {
	Action          string   `json:"action"`
	Names           []string `json:"names"`
	SubscriptionIDs []string `json:"subscription_ids"`
	ServerID        string   `json:"server_id"`
	CombinationID   string   `json:"combination_id"`
	TrafficLimit    int64    `json:"traffic_limit"`
	Remark          string   `json:"remark"`
}

type BatchSubscriptionResult struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
