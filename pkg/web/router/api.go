package router

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"my-digital-home/pkg/web/handler"
	"my-digital-home/pkg/web/middleware"
)

func RegisterAPIs(h *server.Hertz) {
	// 初始化Handler实例
	healthHandler := handler.NewHealthCheckHandler()

	// 注册全局中间件（按执行顺序）
	h.Use(
		middleware.RecoveryMiddleware(),            // 最先处理panic
		middleware.LoggerMiddleware(),              // 记录访问日志
		middleware.SecurityCheckMiddleware(10<<20), // 10MB限制
		middleware.TimeoutMiddleware(15),           // 超时15秒
		middleware.CORSMiddleware(),                // CORS跨域
	)

	// 基础接口组（修改点1：使用结构体方法）
	h.GET("/health", healthHandler.AdvancedHealthCheck)

	// 业务接口组（示范案例）
	// apiGroup := h.Group("/api/v1")
	{
		// 示例写法：
		// userHandler := handler.NewUserHandler()
		// apiGroup.POST("/users", userHandler.CreateUser)
	}
}
