package main

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"my-digital-home/pkg/common/config"
	"my-digital-home/pkg/web/router"
)

func main() {
	// 初始化配置
	cfg := config.Load()

	// 创建Hertz实例
	h := server.Default(
		server.WithHostPorts(cfg.Server.Address),
		server.WithHandleMethodNotAllowed(true),
	)

	// 注册路由
	router.RegisterAPIs(h, cfg)

	// 启动服务
	h.Spin()
}
