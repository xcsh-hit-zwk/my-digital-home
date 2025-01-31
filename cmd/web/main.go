package main

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"my-digital-home/pkg/common/config"
	dao "my-digital-home/pkg/core/user/repository/dao/impl"
	"my-digital-home/pkg/web/router"
)

func main() {
	// 初始化配置
	cfg := config.Load()

	// 初始化数据库连接
	db, err := cfg.InitDB()
	if err != nil {
		panic("Failed to initialize database: " + err.Error())
	}

	// 注入到DAO层
	dao.NewGormUserRepository(db)

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
