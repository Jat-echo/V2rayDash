package model

import "time"

type Subscription struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	UUID         string    `json:"uuid"`
	Enable       bool      `json:"enable"`
	TrafficLimit int64     `json:"traffic_limit"`
	TrafficUsed  int64     `json:"traffic_used"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SubscriptionWithAccounts struct {
	Subscription
	Accounts []*AccountWithServerInfo `json:"accounts"`
}

type CreateSubscriptionRequest struct {
	Name           string           `json:"name" binding:"required"`
	TrafficLimit   int64            `json:"traffic_limit"`
	AccountMappings []AccountMapping `json:"account_mappings"`
}

type UpdateSubscriptionRequest struct {
	Name            *string           `json:"name"`
	Enable          *bool              `json:"enable"`
	TrafficLimit    *int64             `json:"traffic_limit"`
	AccountMappings *[]AccountMapping  `json:"account_mappings"`
}