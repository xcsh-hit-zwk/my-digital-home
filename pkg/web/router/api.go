package router

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"my-digital-home/pkg/common/config"
	"my-digital-home/pkg/web/handler"
	"my-digital-home/pkg/web/middleware"
)

// RegisterAPIs 注册所有API路由
func RegisterAPIs(h *server.Hertz, cfg *config.Config) {
	// 初始化Handler实例
	healthHandler := handler.NewHealthCheckHandler()
	userHandler := handler.NewUserHandler(cfg)

	// 注册全局中间件（按执行顺序）
	h.Use(
		middleware.RecoveryMiddleware(cfg),
		middleware.LoggerMiddleware(),
		middleware.SecurityCheckMiddleware(cfg.Middleware.Security.MaxBodySize),
		middleware.TimeoutMiddleware(cfg.Middleware.Timeout.RequestTimeout),
		middleware.CORSMiddleware(cfg.Middleware.CORS),
		middleware.RateLimitMiddleware(
			cfg.Middleware.RateLimit.Rate,
			cfg.Middleware.RateLimit.Interval,
		),
	)

	// 基础接口组
	h.GET("/health", healthHandler.AdvancedHealthCheck)

	// 业务接口组
	apiGroup := h.Group("/api/v1")
	{
		// 用户相关接口
		userGroup := apiGroup.Group("/users")
		{
			userGroup.POST("/register", userHandler.Register)
			userGroup.POST("/login", userHandler.Login)

			// 需要身份认证的接口
			userGroup.Use(middleware.JWTAuthMiddleware(&cfg.Middleware.JWT))
			userGroup.PUT("/password", userHandler.ChangePassword)
		}
	}
}
