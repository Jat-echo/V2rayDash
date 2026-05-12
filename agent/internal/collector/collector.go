package collector

import (
	"fmt"
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

	v2rayStatus := c.checkV2ray()

	return &model.NodeStatus{
		CPUPercent:    cpu,
		MemoryPercent: mem,
		DiskPercent:   disk,
		V2rayStatus:   v2rayStatus,
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
			if len(fields) >= 3 {
				total, _ := strconv.ParseFloat(fields[1], 64)
				used, _ := strconv.ParseFloat(fields[2], 64)
				if total > 0 {
					return (used / total) * 100, nil
				}
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

func (c *Collector) checkV2ray() string {
	if runtime.GOOS == "linux" {
		cmd := exec.Command("systemctl", "is-active", "v2ray")
		output, _ := cmd.Output()
		if strings.TrimSpace(string(output)) == "active" {
			return "running"
		}
	}
	return "stopped"
}