package collector

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

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

	return &model.NodeStatus{
		CPUPercent:     cpu,
		MemoryPercent:  mem,
		DiskPercent:    disk,
		BandwidthIn:    bandwidthIn,
		BandwidthOut:   bandwidthOut,
		V2rayStatus:    v2rayStatus,
	}, nil
}

func (c *Collector) getCPUUsage() (float64, error) {
	if runtime.GOOS == "linux" {
		return c.getLinuxCPU()
	}
	return 0, nil
}

func (c *Collector) getLinuxCPU() (float64, error) {
	cmd := exec.Command("top", "-bn1")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Cpu(s)") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "id," || p == "id" {
					if i > 0 {
						idle, _ := strconv.ParseFloat(strings.ReplaceAll(parts[i-1], ",", ""), 64)
						return 100 - idle, nil
					}
				}
			}
		}
	}
	return 0, fmt.Errorf("could not parse CPU usage")
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