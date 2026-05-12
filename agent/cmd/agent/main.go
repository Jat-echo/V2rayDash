package main

import (
	"flag"
	"log"
	"time"

	"v2ray-dash/agent/internal/collector"
	"v2ray-dash/agent/internal/config"
	"v2ray-dash/agent/internal/reporter"
)

func main() {
	configPath := flag.String("config", "/etc/v2ray-agent/agent.json", "Agent config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Agent starting, server_id: %s, control_center: %s", cfg.ServerID, cfg.ControlCenterURL)

	reporterClient := reporter.New(cfg.ControlCenterURL, cfg.ServerID, cfg.PSK)
	col := collector.New()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 立即执行一次
	reportStatus(cfg.ServerID, reporterClient, col)

	for range ticker.C {
		reportStatus(cfg.ServerID, reporterClient, col)
	}
}

func reportStatus(serverID string, client *reporter.Client, col *collector.Collector) {
	status, err := col.Collect()
	if err != nil {
		log.Printf("Collect error: %v", err)
		return
	}

	status.ServerID = serverID

	if err := client.ReportStatus(status); err != nil {
		log.Printf("Report error: %v", err)
	}
}