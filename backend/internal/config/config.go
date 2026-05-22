package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerPort        string
	DatabaseURL       string
	JWTSecret         string
	ControlCenterURL  string
	InstallScriptPath string
}

func Load() *Config {
	return &Config{
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://localhost:5432/v2ray_dash?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", "change-me-in-production"),
		ControlCenterURL:  getEnv("CONTROL_CENTER_URL", "http://localhost:8080"),
		InstallScriptPath: getEnv("INSTALL_SCRIPT_PATH", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) Validate() error {
	if c.JWTSecret == "" || c.JWTSecret == "change-me-in-production" {
		return fmt.Errorf("JWT_SECRET must be set")
	}
	return nil
}