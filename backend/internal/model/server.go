package model

import (
	"time"
)

type Server struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	IP         string    `json:"ip"`
	SSHPort    int       `json:"ssh_port"`
	SSHUser    string    `json:"ssh_user"`
	SSHKeyType string    `json:"ssh_key_type"` // "key" or "password"
	SSHKey     string    `json:"-"`            // 敏感字段不暴露
	SSHPassword string   `json:"-"`            // 敏感字段不暴露
	Tags       []string  `json:"tags"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateServerRequest struct {
	Name        string   `json:"name" binding:"required"`
	IP          string   `json:"ip" binding:"required"`
	SSHPort     int      `json:"ssh_port"`
	SSHUser     string   `json:"ssh_user"`
	SSHKeyType  string   `json:"ssh_key_type"` // "key" or "password", default "key"
	SSHKey      string   `json:"ssh_key"`
	SSHPassword string   `json:"ssh_password"`
	Tags        []string `json:"tags"`
}

type UpdateServerRequest struct {
	Name        *string  `json:"name"`
	SSHPort     *int     `json:"ssh_port"`
	SSHUser     *string  `json:"ssh_user"`
	SSHKeyType  *string  `json:"ssh_key_type"`
	SSHKey      *string  `json:"ssh_key"`
	SSHPassword *string  `json:"ssh_password"`
	Tags        []string `json:"tags"`
}