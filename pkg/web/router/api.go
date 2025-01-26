package router

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"my-digital-home/pkg/common/config"
	"my-digital-home/pkg/web/handler"
	"my-digital-home/pkg/web/middleware"
	"time"
)

// RegisterAPIs 注册所有API路由
func RegisterAPIs(h *server.Hertz, cfg *config.Config) {
	// 初始化Handler实例
	healthHandler := handler.NewHealthCheckHandler()

	// 注册全局中间件（按执行顺序）
	h.Use(
		middleware.RecoveryMiddleware(cfg),              // 最先处理panic，需要传入配置
		middleware.LoggerMiddleware(),                   // 记录访问日志
		middleware.SecurityCheckMiddleware(10<<20),      // 10MB限制
		middleware.TimeoutMiddleware(15),                // 超时15秒
		middleware.CORSMiddleware(),                     // CORS跨域
		middleware.RateLimitMiddleware(10, time.Second), // 限流10次/秒，需要指定时间间隔
	)

	// 基础接口组
	h.GET("/health", healthHandler.AdvancedHealthCheck)

	// 业务接口组（示范案例）
	// apiGroup := h.Group("/api/v1")
	{
		// 示例写法：
		// userHandler := handler.NewUserHandler()
		// apiGroup.POST("/users", userHandler.CreateUser)
	}
}
