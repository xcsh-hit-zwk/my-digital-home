# 本地开发
├── config.json           # 本地配置文件
└── main.go

# 阿里云部署
/etc/my-digital-home/
└── config.json          # 生产环境配置文件

# 本地开发启动
go run main.go

# 使用自定义配置文件启动
APP_CONFIG=/path/to/config.json go run main.go

# 使用环境变量覆盖配置
APP_ENV=production SERVER_ADDR=:9090 go run main.go


# 1. 在服务器创建配置目录
mkdir -p /etc/my-digital-home/

# 2. 上传配置文件
scp config.json root@your-server:/etc/my-digital-home/

# 3. 设置环境变量（可以放在systemd服务文件中）
cat > /etc/systemd/system/my-digital-home.service << EOF
[Unit]
Description=My Digital Home Service
After=network.target

[Service]
Environment=APP_ENV=production
Environment=APP_CONFIG=/etc/my-digital-home/config.json
ExecStart=/usr/local/bin/my-digital-home
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# 4. 启动服务
systemctl daemon-reload
systemctl enable my-digital-home
systemctl start my-digital-home
