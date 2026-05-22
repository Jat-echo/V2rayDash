package model

import "time"

type SubscriptionAccount struct {
	ID             string    `json:"id"`
	SubscriptionID string    `json:"subscription_id"`
	AccountID      string    `json:"account_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type AccountMapping struct {
	ServerID   string `json:"server_id"`
	AccountID  string `json:"account_id"`
	AutoCreate bool   `json:"auto_create"`
}

type AccountWithServerInfo struct {
	Account
	ServerName string `json:"server_name"`
	ServerIP   string `json:"server_ip"`
}