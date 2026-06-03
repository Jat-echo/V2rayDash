package handler

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/repository"
)

// cpuCache 避免每次 HTTP 请求都阻塞 300ms 采样 CPU
var (
	cpuCacheMu    sync.RWMutex
	cpuCacheValue float64
)

func init() {
	go func() {
		for {
			if s1, err := readLocalCPUStat(); err == nil {
				time.Sleep(500 * time.Millisecond)
				if s2, err := readLocalCPUStat(); err == nil {
					idle1 := s1.idle + s1.iowait
					idle2 := s2.idle + s2.iowait
					total1 := s1.user + s1.nice + s1.system + s1.idle + s1.iowait + s1.irq + s1.softirq + s1.steal
					total2 := s2.user + s2.nice + s2.system + s2.idle + s2.iowait + s2.irq + s2.softirq + s2.steal
					if diff := total2 - total1; diff > 0 {
						v := (1 - float64(idle2-idle1)/float64(diff)) * 100
						cpuCacheMu.Lock()
						cpuCacheValue = v
						cpuCacheMu.Unlock()
					}
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

type SettingHandler struct {
	repo *repository.SettingRepository
}

func NewSettingHandler(repo *repository.SettingRepository) *SettingHandler {
	return &SettingHandler{repo: repo}
}

// fetchPublicIP attempts to get public IP from multiple services
func fetchPublicIP() (string, error) {
	// Try multiple services in order of preference
	services := []string{
		"https://api.ipify.org?format=text",
		"https://icanhazip.com",
		"https://ifconfig.me/ip",
		"http://checkip.amazonaws.com",
		"https://api.my-ip.io/v1/ip.json",
	}

	for _, service := range services {
		resp, err := http.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		// Read IP address
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ip := strings.TrimSpace(string(body))
		// Validate IP format (basic check)
		if len(ip) > 6 && len(ip) < 45 {
			return ip, nil
		}
	}

	return "", nil
}

func (h *SettingHandler) GetPublicIP(c *gin.Context) {
	ip, err := fetchPublicIP()
	if err != nil || ip == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get public IP"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ip": ip})
}

func (h *SettingHandler) GetPublicURL(c *gin.Context) {
	setting, err := h.repo.Get("public_url")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"value": "http://localhost:8080"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"value": setting.Value})
}

type localCPUStat struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

func readLocalCPUStat() (localCPUStat, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return localCPUStat{}, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			return localCPUStat{}, fmt.Errorf("unexpected format")
		}
		p := func(s string) uint64 { v, _ := strconv.ParseUint(s, 10, 64); return v }
		st := localCPUStat{
			user: p(fields[1]), nice: p(fields[2]), system: p(fields[3]),
			idle: p(fields[4]), iowait: p(fields[5]), irq: p(fields[6]),
			softirq: p(fields[7]),
		}
		if len(fields) > 8 {
			st.steal = p(fields[8])
		}
		return st, nil
	}
	return localCPUStat{}, fmt.Errorf("cpu line not found")
}

func (h *SettingHandler) GetSystemStatus(c *gin.Context) {
	// CPU: 读后台缓存值（避免在 HTTP handler 中阻塞采样）
	cpuCacheMu.RLock()
	cpuPercent := cpuCacheValue
	cpuCacheMu.RUnlock()

	// Memory: 解析 /proc/meminfo
	var memTotal, memAvail uint64
	if mf, err := os.Open("/proc/meminfo"); err == nil {
		defer mf.Close()
		sc := bufio.NewScanner(mf)
		for sc.Scan() {
			line := sc.Text()
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			val, _ := strconv.ParseUint(fields[1], 10, 64)
			switch fields[0] {
			case "MemTotal:":
				memTotal = val
			case "MemAvailable:":
				memAvail = val
			}
		}
	}
	var memPercent float64
	var memUsedMB, memTotalMB uint64
	if memTotal > 0 {
		memUsedMB = (memTotal - memAvail) / 1024
		memTotalMB = memTotal / 1024
		memPercent = float64(memTotal-memAvail) / float64(memTotal) * 100
	}

	// Disk: syscall.Statfs on /
	var diskPercent float64
	var diskUsedGB, diskTotalGB float64
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err == nil {
		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bfree * uint64(stat.Bsize)
		used := total - free
		diskTotalGB = float64(total) / (1024 * 1024 * 1024)
		diskUsedGB = float64(used) / (1024 * 1024 * 1024)
		if total > 0 {
			diskPercent = float64(used) / float64(total) * 100
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"cpu_percent":   cpuPercent,
		"mem_percent":   memPercent,
		"mem_used_mb":   memUsedMB,
		"mem_total_mb":  memTotalMB,
		"disk_percent":  diskPercent,
		"disk_used_gb":  diskUsedGB,
		"disk_total_gb": diskTotalGB,
	})
}

func (h *SettingHandler) UpdatePublicURL(c *gin.Context) {
	var req struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Update("public_url", req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update setting"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated", "value": req.Value})
}
