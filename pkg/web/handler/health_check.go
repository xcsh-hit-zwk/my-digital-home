package handler

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"time"
)

type HealthCheckHandler struct {
}

func NewHealthCheckHandler() *HealthCheckHandler {
	return &HealthCheckHandler{}
}

type HealthStatus struct {
	Status     string            `json:"status"`
	Timestamp  time.Time         `json:"timestamp"`
	Components []ComponentStatus `json:"components,omitempty"`
}

// 方案2：启用关键组件标签判断
type ComponentStatus struct {
	Name    string        `json:"name"`
	Status  string        `json:"status"`
	IsCore  bool          `json:"is_core"` // 新增关键组件标识
	Latency time.Duration `json:"latency,omitempty"`
	Error   string        `json:"error,omitempty"`
}

var startupTime = time.Now()

// AdvancedHealthCheck 增强的健康检查接口
func (h *HealthCheckHandler) AdvancedHealthCheck(ctx context.Context, c *app.RequestContext) {
	status := HealthStatus{
		Status:     "healthy",
		Timestamp:  time.Now().UTC(),
		Components: []ComponentStatus{
			//checkDatabase(),
			//checkRedis(),
			//checkExternalService(),
		},
	}

	if hasCriticalErrors(status.Components) {
		status.Status = "degraded"
		c.JSON(503, status)
		return
	}

	c.JSON(200, status)
}

func hasCriticalErrors(components []ComponentStatus) bool {
	for _, comp := range components {
		// 核心组件状态异常或任意组件发生严重错误
		if (comp.IsCore && comp.Status != "ok") || comp.Status == "critical" {
			return true
		}
	}
	return false
}
