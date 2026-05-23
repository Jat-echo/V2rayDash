package model

import (
	"time"
)

type Account struct {
	ID            string    `json:"id"`
	ServerID      string    `json:"server_id"`
	UUID          string    `json:"uuid"`
	Email         string    `json:"email"`
	Protocols     []string  `json:"protocols"`
	Enabled       bool      `json:"enabled"`
	TrafficLimit  int64     `json:"traffic_limit"`
	TrafficUsed   int64     `json:"traffic_used"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateAccountRequest struct {
	ServerID  string   `json:"server_id" binding:"required"`
	UUID     string   `json:"uuid"`
	Email    string   `json:"email" binding:"required"`
	Protocols []string `json:"protocols" binding:"required"`
}

type UpdateAccountRequest struct {
	Email        *string  `json:"email"`
	UUID         *string  `json:"uuid"`
	Protocols    []string `json:"protocols"`
	Enabled      *bool    `json:"enabled"`
	TrafficLimit *int64   `json:"traffic_limit"`
}