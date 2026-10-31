package model

import (
	"time"
)

type Server struct {
	Role              string       `json:"role"`
	Node              *ManagedNode `json:"node,omitempty"`
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	IP                string       `json:"ip"`
	CountryCode       string       `json:"country_code"` // ISO 3166-1 alpha-2, e.g. "JP"
	SSHPort           int          `json:"ssh_port"`
	SSHJumpServerID   string       `json:"ssh_jump_server_id"`
	SSHUser           string       `json:"ssh_user"`
	SSHKeyType        string       `json:"ssh_key_type"` // "key" or "password"
	SSHKey            string       `json:"-"`            // 敏感字段不暴露
	SSHPassword       string       `json:"-"`            // 敏感字段不暴露
	Tags              []string     `json:"tags"`
	Status            string       `json:"status"`
	RealityEnabled    bool         `json:"reality_enabled"`
	RealityServerName string       `json:"reality_server_name"`
	RealityPublicKey  string       `json:"reality_public_key"`
	RealityPort       int          `json:"reality_port"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

type CreateServerRequest struct {
	Role              string   `json:"role" binding:"omitempty,oneof=exit relay"`
	Name              string   `json:"name" binding:"required"`
	IP                string   `json:"ip" binding:"required"`
	CountryCode       string   `json:"country_code"`
	SSHPort           int      `json:"ssh_port"`
	SSHJumpServerID   string   `json:"ssh_jump_server_id"`
	SSHUser           string   `json:"ssh_user"`
	SSHKeyType        string   `json:"ssh_key_type"`
	SSHKey            string   `json:"ssh_key"`
	SSHPassword       string   `json:"ssh_password"`
	Tags              []string `json:"tags"`
	RealityEnabled    bool     `json:"reality_enabled"`
	RealityServerName string   `json:"reality_server_name"`
	RealityPublicKey  string   `json:"reality_public_key"`
	RealityPort       int      `json:"reality_port"`
}

type UpdateServerRequest struct {
	Role              *string  `json:"role" binding:"omitempty,oneof=exit relay"`
	Name              *string  `json:"name"`
	IP                *string  `json:"ip"`
	CountryCode       *string  `json:"country_code"`
	SSHPort           *int     `json:"ssh_port"`
	SSHJumpServerID   *string  `json:"ssh_jump_server_id"`
	SSHUser           *string  `json:"ssh_user"`
	SSHKeyType        *string  `json:"ssh_key_type"`
	SSHKey            *string  `json:"ssh_key"`
	SSHPassword       *string  `json:"ssh_password"`
	Tags              []string `json:"tags"`
	RealityEnabled    *bool    `json:"reality_enabled"`
	RealityServerName *string  `json:"reality_server_name"`
	RealityPublicKey  *string  `json:"reality_public_key"`
	RealityPort       *int     `json:"reality_port"`
}
