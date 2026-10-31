package main

import (
	"flag"
	"log"
	"path/filepath"
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
	col.Service = cfg.CoreService
	col.Cumulative = cfg.CoreService != ""
	statePath := cfg.StatePath
	if statePath == "" {
		statePath = filepath.Join(filepath.Dir(*configPath), "pending-heartbeats.json")
	}
	var queue *reporter.Queue
	if col.Cumulative {
		queue, err = reporter.OpenQueue(statePath)
		if err != nil {
			log.Fatalf("Failed to load pending heartbeats: %v", err)
		}
	}

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// 立即执行一次
	reportStatus(cfg.ServerID, reporterClient, col, queue)

	for range ticker.C {
		reportStatus(cfg.ServerID, reporterClient, col, queue)
	}
}

func reportStatus(serverID string, client *reporter.Client, col *collector.Collector, queue *reporter.Queue) {
	status, err := col.Collect()
	if err != nil {
		log.Printf("Collect error: %v", err)
		return
	}

	status.ServerID = serverID
	if queue == nil {
		if err := client.ReportStatus(status); err != nil {
			log.Printf("Report error: %v", err)
		}
		return
	}

	if err := queue.Enqueue(status); err != nil {
		log.Printf("Persist heartbeat error: %v", err)
		return
	}
	if err := queue.Flush(client.ReportStatus); err != nil {
		log.Printf("Report error: %v", err)
	}
}
