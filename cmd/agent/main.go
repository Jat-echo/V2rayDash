package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Config struct {
	ServerID      string `json:"server_id"`
	ControlCenter string `json:"control_center"`
}

type HeartbeatRequest struct {
	ServerID     string  `json:"server_id"`
	CPUPercent   float64 `json:"cpu_percent"`
	MemPercent   float64 `json:"mem_percent"`
	DiskPercent  float64 `json:"disk_percent"`
	BandwidthIn  int64   `json:"bandwidth_in"`
	BandwidthOut int64   `json:"bandwidth_out"`
	V2rayStatus  string  `json:"v2ray_status"`
}

type StatusResponse struct {
	ServerID      string  `json:"server_id"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	BandwidthIn   int64   `json:"bandwidth_in"`
	BandwidthOut  int64   `json:"bandwidth_out"`
	V2rayStatus   string  `json:"v2ray_status"`
	ReportedAt    string  `json:"reported_at"`
}

func main() {
	// 从控制中心获取配置
	config := getConfig()

	// 定期发送心跳
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			status := collectStatus(config.ServerID)
			sendHeartbeat(config.ControlCenter, status)
		}
	}
}

func getConfig() Config {
	// 实际应该从文件读取或通过 API 获取
	// 这里简化处理
	return Config{
		ServerID:      "your-server-id",
		ControlCenter: "http://localhost:8080",
	}
}

func collectStatus(serverID string) HeartbeatRequest {
	// 收集系统状态
	cpu := getCPUUsage()
	mem := getMemoryUsage()
	disk := getDiskUsage()
	bandwidth := getBandwidth()
	v2ray := checkV2rayStatus()

	return HeartbeatRequest{
		ServerID:     serverID,
		CPUPercent:   cpu,
		MemPercent:   mem,
		DiskPercent:  disk,
		BandwidthIn:  bandwidth.in,
		BandwidthOut: bandwidth.out,
		V2rayStatus:  v2ray,
	}
}

func sendHeartbeat(center string, status HeartbeatRequest) {
	url := fmt.Sprintf("%s/api/agent/heartbeat", center)
	body, _ := json.Marshal(status)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Heartbeat failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("[%s] Heartbeat sent successfully\n", time.Now().Format("15:04:05"))
	}
}

func getCPUUsage() float64 {
	// Linux: 读取 /proc/stat
	// 示例实现，实际应解析 /proc/stat
	var usage float64 = 25.5
	return usage
}

func getMemoryUsage() float64 {
	// Linux: 读取 /proc/meminfo
	var usage float64 = 45.2
	return usage
}

func getDiskUsage() float64 {
	// Linux: 使用 syscall.Statfs
	var usage float64 = 67.8
	return usage
}

func getBandwidth() struct{ in, out int64 } {
	// 读取 /proc/net/dev 计算流量
	return struct{ in, out int64 }{1024000, 2048000}
}

func checkV2rayStatus() string {
	// 检查 v2ray/xray 进程是否运行
	return "running"
}

// StartWebServer 启动本地状态查询接口 (供控制中心调用)
func StartWebServer() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/status", func(c *gin.Context) {
		status := collectStatus("local")
		c.JSON(http.StatusOK, status)
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	go r.Run(":9090")
}