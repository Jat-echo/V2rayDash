package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"v2ray-dash/agent/internal/model"
)

type Collector struct{}

func New() *Collector {
	return &Collector{}
}

func (c *Collector) Collect() (*model.NodeStatus, error) {
	cpu, err := c.getCPUUsage()
	if err != nil {
		cpu = 0
	}

	mem, err := c.getMemoryUsage()
	if err != nil {
		mem = 0
	}

	disk, err := c.getDiskUsage()
	if err != nil {
		disk = 0
	}

	// 新增：获取带宽
	bandwidthIn, bandwidthOut, _ := c.getBandwidth()

	v2rayStatus := c.checkV2ray()
	userTraffic := c.getXrayUserTraffic()

	return &model.NodeStatus{
		CPUPercent:       cpu,
		MemoryPercent:    mem,
		DiskPercent:      disk,
		BandwidthIn:      bandwidthIn,
		BandwidthOut:     bandwidthOut,
		V2rayStatus:      v2rayStatus,
		UserTrafficStats: userTraffic,
	}, nil
}

func (c *Collector) getCPUUsage() (float64, error) {
	if runtime.GOOS == "linux" {
		return c.getLinuxCPU()
	}
	return 0, nil
}

type cpuStat struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

func readProcStat() (cpuStat, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuStat{}, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			return cpuStat{}, fmt.Errorf("unexpected /proc/stat format")
		}
		parse := func(s string) uint64 { v, _ := strconv.ParseUint(s, 10, 64); return v }
		return cpuStat{
			user:    parse(fields[1]),
			nice:    parse(fields[2]),
			system:  parse(fields[3]),
			idle:    parse(fields[4]),
			iowait:  parse(fields[5]),
			irq:     parse(fields[6]),
			softirq: parse(fields[7]),
			steal:   func() uint64 { if len(fields) > 8 { return parse(fields[8]) }; return 0 }(),
		}, nil
	}
	return cpuStat{}, fmt.Errorf("cpu line not found in /proc/stat")
}

func (c *Collector) getLinuxCPU() (float64, error) {
	s1, err := readProcStat()
	if err != nil {
		return 0, err
	}
	time.Sleep(500 * time.Millisecond)
	s2, err := readProcStat()
	if err != nil {
		return 0, err
	}

	idle1 := s1.idle + s1.iowait
	idle2 := s2.idle + s2.iowait
	total1 := s1.user + s1.nice + s1.system + s1.idle + s1.iowait + s1.irq + s1.softirq + s1.steal
	total2 := s2.user + s2.nice + s2.system + s2.idle + s2.iowait + s2.irq + s2.softirq + s2.steal

	totalDiff := total2 - total1
	if totalDiff == 0 {
		return 0, nil
	}
	idleDiff := idle2 - idle1
	return (1 - float64(idleDiff)/float64(totalDiff)) * 100, nil
}

func (c *Collector) getMemoryUsage() (float64, error) {
	if runtime.GOOS == "linux" {
		cmd := exec.Command("free", "-m")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}

		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			// 格式: Mem: total used free shared buff/cache available
			// 或: Mem: total used free shared buff/cache available (with Mem: as part of line)
			// 找到 total 和 available 字段
			var total, available float64
			if fields[0] == "Mem:" {
				// 单位是 MB
				total, _ = strconv.ParseFloat(fields[1], 64)
				available, _ = strconv.ParseFloat(fields[6], 64) // available 是第7个字段
			} else {
				// bytes 格式或其他
				total, _ = strconv.ParseFloat(fields[0], 64)
				available, _ = strconv.ParseFloat(fields[6], 64)
			}
			if total > 0 {
				// 使用 (total - available) / total * 100 来计算已用内存百分比
				used := total - available
				return (used / total) * 100, nil
			}
		}
	}
	return 0, nil
}

func (c *Collector) getDiskUsage() (float64, error) {
	if runtime.GOOS == "linux" {
		cmd := exec.Command("df", "-h", "/")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}

		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 5 {
				usage := strings.TrimSuffix(fields[4], "%")
				return strconv.ParseFloat(usage, 64)
			}
		}
	}
	return 0, nil
}

func (c *Collector) getBandwidth() (int64, int64, error) {
	if runtime.GOOS != "linux" {
		return 0, 0, nil
	}

	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(data), "\n")
	var totalRx, totalTx int64

	for _, line := range lines[2:] { // 跳过前两行表头
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		// 格式: eth0: rx_bytes rx_packets ... tx_bytes tx_packets
		interfaceName := strings.TrimSuffix(fields[0], ":")
		if interfaceName == "lo" {
			continue
		}

		rx, _ := strconv.ParseInt(fields[1], 10, 64)
		tx, _ := strconv.ParseInt(fields[9], 10, 64)

		totalRx += rx
		totalTx += tx
	}

	return totalRx, totalTx, nil
}

func (c *Collector) checkV2ray() string {
	if runtime.GOOS == "linux" {
		// 检查多种可能的服务名称
		serviceNames := []string{"xray", "v2ray", "sing-box"}
		for _, name := range serviceNames {
			cmd := exec.Command("systemctl", "is-active", name)
			output, _ := cmd.Output()
			if strings.TrimSpace(string(output)) == "active" {
				return "running"
			}
		}
		// 备用：检查进程是否存在
		processNames := []string{"xray", "v2ray", "sing-box"}
		for _, name := range processNames {
			cmd := exec.Command("pgrep", "-f", name)
			output, _ := cmd.Output()
			if len(output) > 0 {
				return "running"
			}
		}
	}
	return "stopped"
}

func (c *Collector) getXrayUserTraffic() []model.UserTrafficStat {
	if runtime.GOOS != "linux" {
		return nil
	}
	// 查询 xray 用户流量并重置计数，获取自上次查询以来的增量
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "xray", "api", "statsquery",
		"-server", "127.0.0.1:10085",
		"-pattern", "user",
		"-reset")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var result struct {
		Stat []struct {
			Name  string `json:"name"`
			Value int64  `json:"value"`
		} `json:"stat"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil
	}

	userMap := make(map[string]*model.UserTrafficStat)
	for _, s := range result.Stat {
		// 格式: user>>>email>>>traffic>>>uplink|downlink
		parts := strings.SplitN(s.Name, ">>>", 4)
		if len(parts) != 4 || parts[0] != "user" {
			continue
		}
		email := parts[1]
		direction := parts[3]
		value := s.Value
		if value <= 0 {
			continue
		}
		if _, ok := userMap[email]; !ok {
			userMap[email] = &model.UserTrafficStat{Email: email}
		}
		if direction == "uplink" {
			userMap[email].Upload += value
		} else if direction == "downlink" {
			userMap[email].Download += value
		}
	}

	stats := make([]model.UserTrafficStat, 0, len(userMap))
	for _, s := range userMap {
		stats = append(stats, *s)
	}
	return stats
}