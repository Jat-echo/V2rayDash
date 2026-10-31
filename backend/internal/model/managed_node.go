package model

import "time"

// ManagedNode describes an adopted Xray installation. Private Reality keys stay on the node.
type ManagedNode struct {
	ServerID   string    `json:"server_id"`
	Role       string    `json:"role"`
	Layout     string    `json:"layout"`
	InboundTag string    `json:"inbound_tag"`
	ShortID    string    `json:"short_id"`
	EgressIP   string    `json:"egress_ip"`
	State      string    `json:"state"`
	LastError  string    `json:"last_error"`
	ConfigHash string    `json:"config_hash"`
	SSHHostKey string    `json:"-"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type RelayRoute struct {
	AccountID            string `json:"account_id"`
	ExitAccountID        string `json:"exit_account_id"`
	AppliedExitAccountID string `json:"applied_exit_account_id"`
	Revision             int64  `json:"revision"`
	AppliedRevision      int64  `json:"applied_revision"`
}
