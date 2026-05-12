package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	ServerID        string `json:"server_id"`
	ControlCenterURL string `json:"control_center_url"`
	PSK             string `json:"psk"` // Pre-shared key for authentication
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}