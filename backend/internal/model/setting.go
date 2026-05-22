package model

import "time"

type SystemSetting struct {
	ID          string    `json:"id"`
	Value       string    `json:"value"`
	Description string    `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpdateSettingRequest struct {
	Value string `json:"value" binding:"required"`
}
