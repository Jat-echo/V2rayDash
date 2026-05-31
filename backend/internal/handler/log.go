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
	// Calculate bandwidth delta
	var baseIn, baseOut int64
	if len(statuses) > 0 {
		baseIn = statuses[0].BandwidthIn
		baseOut = statuses[0].BandwidthOut
	}

	cpu := make([]model.MetricPoint, 0, len(statuses))
	memory := make([]model.MetricPoint, 0, len(statuses))
	disk := make([]model.MetricPoint, 0, len(statuses))
	bandwidthIn := make([]model.BandwidthPoint, 0, len(statuses))
	bandwidthOut := make([]model.BandwidthPoint, 0, len(statuses))

	for _, s := range statuses {
		cpu = append(cpu, model.MetricPoint{Time: s.ReportedAt, Value: s.CPUPercent})
		memory = append(memory, model.MetricPoint{Time: s.ReportedAt, Value: s.MemoryPercent})
		disk = append(disk, model.MetricPoint{Time: s.ReportedAt, Value: s.DiskPercent})
		bandwidthIn = append(bandwidthIn, model.BandwidthPoint{Time: s.ReportedAt, Value: s.BandwidthIn - baseIn})
		bandwidthOut = append(bandwidthOut, model.BandwidthPoint{Time: s.ReportedAt, Value: s.BandwidthOut - baseOut})
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
	}
}