package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
)

type LogHandler struct {
	logRepo *repository.LogRepository
}

func NewLogHandler(logRepo *repository.LogRepository) *LogHandler {
	return &LogHandler{logRepo: logRepo}
}

func (h *LogHandler) ListOperationLogs(c *gin.Context) {
	filter := &model.OperationLogFilter{}

	logs, err := h.logRepo.List(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

func (h *LogHandler) ListNodeStatuses(c *gin.Context) {
	timeRange := c.DefaultQuery("time_range", "1h")
	serverID := c.Query("server_id")

	statuses, err := h.logRepo.GetNodeStatusesByTimeRange(serverID, timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(statuses) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	// Group by server_id
	grouped := make(map[string][]*model.NodeStatus)
	for _, s := range statuses {
		grouped[s.ServerID] = append(grouped[s.ServerID], s)
	}

	var responses []model.NodeStatusResponse
	for sid, ss := range grouped {
		resp := buildNodeStatusResponse(sid, ss)
		responses = append(responses, resp)
	}

	c.JSON(http.StatusOK, responses)
}

func buildNodeStatusResponse(serverID string, statuses []*model.NodeStatus) model.NodeStatusResponse {
	cpu := make([]model.MetricPoint, 0, len(statuses))
	memory := make([]model.MetricPoint, 0, len(statuses))
	disk := make([]model.MetricPoint, 0, len(statuses))
	bandwidthIn := make([]model.BandwidthPoint, 0, len(statuses))
	bandwidthOut := make([]model.BandwidthPoint, 0, len(statuses))

	// Calculate bandwidth delta (change between consecutive points)
	var prevIn, prevOut int64
	for i, s := range statuses {
		cpu = append(cpu, model.MetricPoint{Time: s.ReportedAt, Value: s.CPUPercent})
		memory = append(memory, model.MetricPoint{Time: s.ReportedAt, Value: s.MemoryPercent})
		disk = append(disk, model.MetricPoint{Time: s.ReportedAt, Value: s.DiskPercent})

		// Calculate delta between consecutive points; negative delta means counter reset (reboot), treat as 0
		var deltaIn, deltaOut int64
		if i > 0 {
			deltaIn = s.BandwidthIn - prevIn
			deltaOut = s.BandwidthOut - prevOut
			if deltaIn < 0 {
				deltaIn = 0
			}
			if deltaOut < 0 {
				deltaOut = 0
			}
		}
		bandwidthIn = append(bandwidthIn, model.BandwidthPoint{Time: s.ReportedAt, Value: deltaIn})
		bandwidthOut = append(bandwidthOut, model.BandwidthPoint{Time: s.ReportedAt, Value: deltaOut})
		prevIn = s.BandwidthIn
		prevOut = s.BandwidthOut
	}

	// Count xray restart events: transitions stopped→running after the first point
	v2rayRestarts := 0
	for i := 1; i < len(statuses); i++ {
		if statuses[i-1].V2rayStatus != "running" && statuses[i].V2rayStatus == "running" {
			v2rayRestarts++
		}
	}

	latest := statuses[len(statuses)-1]

	return model.NodeStatusResponse{
		ServerID: serverID,
		Metrics: model.NodeStatusMetrics{
			CPU:          cpu,
			Memory:       memory,
			Disk:         disk,
			BandwidthIn:  bandwidthIn,
			BandwidthOut: bandwidthOut,
		},
		Current: &model.NodeStatusCurrent{
			CPUPercent:    latest.CPUPercent,
			MemoryPercent: latest.MemoryPercent,
			DiskPercent:   latest.DiskPercent,
			BandwidthIn:   latest.BandwidthIn,
			BandwidthOut:  latest.BandwidthOut,
			V2rayStatus:   latest.V2rayStatus,
			ReportedAt:    latest.ReportedAt,
		},
		V2rayRestarts: v2rayRestarts,
	}
}