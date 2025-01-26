package config

import (
	"encoding/json"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

type ServerConfig struct {
	Address string `json:"address"`
}

type SecurityConfig struct {
	MaxBodySize    int64    `json:"maxBodySize"` // 单位：字节
	AllowedHosts   []string `json:"allowedHosts"`
	AllowedMethods []string `json:"allowedMethods"`
}

type TimeoutConfig struct {
	RequestTimeout int `json:"requestTimeout"` // 单位：秒
}

type CORSConfig struct {
	AllowOrigins     []string      `json:"allowOrigins"`
	AllowMethods     []string      `json:"allowMethods"`
	AllowHeaders     []string      `json:"allowHeaders"`
	ExposeHeaders    []string      `json:"exposeHeaders"`
	AllowCredentials bool          `json:"allowCredentials"`
	MaxAge           time.Duration `json:"maxAge"`
	TrustedDomains   []string      `json:"trustedDomains"`
}

type RateLimitConfig struct {
	Rate     int           `json:"rate"`
	Interval time.Duration `json:"interval"`
}

type MiddlewareConfig struct {
	Security  SecurityConfig  `json:"security"`
	Timeout   TimeoutConfig   `json:"timeout"`
	CORS      CORSConfig      `json:"cors"`
	RateLimit RateLimitConfig `json:"rateLimit"`
}

type Config struct {
	Server     ServerConfig     `json:"server"`
	Middleware MiddlewareConfig `json:"middleware"`
	Env        string           `json:"env"` // 环境标识
}

// 默认配置
var defaultConfig = Config{
	Server: ServerConfig{
		Address: ":8080",
	},
	Middleware: MiddlewareConfig{
		Security: SecurityConfig{
			MaxBodySize:    10 << 20, // 10MB
			AllowedMethods: []string{"GET", "POST", "PUT"},
		},
		Timeout: TimeoutConfig{
			RequestTimeout: 15,
		},
		CORS: CORSConfig{
			AllowOrigins:     []string{"http://localhost:3000"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Content-Type", "Authorization", "X-Requested-With"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
			TrustedDomains:   []string{".dev.your-company.com"},
		},
		RateLimit: RateLimitConfig{
			Rate:     10,
			Interval: time.Second,
		},
	},
	Env: "development",
}

// IsProd 判断当前是否生产环境
func (c *Config) IsProd() bool {
	return c.Env == "production"
}

// Load 加载配置（优先级：环境变量 > 配置文件 > 默认值）
func Load() *Config {
	config := defaultConfig

	// 1. 尝试从配置文件加载
	configPath := getConfigPath()
	if configPath != "" {
		if err := loadFromFile(&config, configPath); err != nil {
			hlog.Warnf("Failed to load config file: %v", err)
		}
	}

	// 2. 从环境变量覆盖
	loadFromEnv(&config)

	return &config
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	// 优先使用环境变量指定的配置文件路径
	if path := os.Getenv("APP_CONFIG"); path != "" {
		return path
	}

	// 依次查找可能的配置文件位置
	searchPaths := []string{
		"./config.json",                    // 当前目录
		"../config.json",                   // 上级目录
		"/etc/my-digital-home/config.json", // 系统配置目录
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// loadFromFile 从文件加载配置
func loadFromFile(config *Config, path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, config)
}

// loadFromEnv 从环境变量加载配置
func loadFromEnv(config *Config) {
	// 服务器配置
	if v := os.Getenv("SERVER_ADDR"); v != "" {
		config.Server.Address = v
	}

	// 环境配置
	if v := os.Getenv("APP_ENV"); v != "" {
		config.Env = v
	}

	// 中间件配置
	if v := os.Getenv("MAX_BODY_SIZE"); v != "" {
		if size, err := strconv.ParseInt(v, 10, 64); err == nil {
			config.Middleware.Security.MaxBodySize = size
		}
	}

	if v := os.Getenv("REQUEST_TIMEOUT"); v != "" {
		if timeout, err := strconv.Atoi(v); err == nil {
			config.Middleware.Timeout.RequestTimeout = timeout
		}
	}

	if v := os.Getenv("RATE_LIMIT"); v != "" {
		if rate, err := strconv.Atoi(v); err == nil {
			config.Middleware.RateLimit.Rate = rate
		}
	}

	// ... 其他环境变量配置项
}
