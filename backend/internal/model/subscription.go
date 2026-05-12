package model

import (
	"time"
)

type Subscription struct {
	ID           string    `json:"id"`
	ServerID     string    `json:"server_id"`
	Name         string    `json:"name"`
	UUID         string    `json:"uuid"`
	Enable       bool      `json:"enable"`
	TrafficLimit int64     `json:"traffic_limit"`
	TrafficUsed  int64     `json:"traffic_used"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateSubscriptionRequest struct {
	ServerID     string `json:"server_id" binding:"required"`
	Name         string `json:"name" binding:"required"`
	TrafficLimit int64  `json:"traffic_limit"`
}

type UpdateSubscriptionRequest struct {
	Name         *string `json:"name"`
	Enable       *bool   `json:"enable"`
	TrafficLimit *int64  `json:"traffic_limit"`
}