package config

import (
	"encoding/json"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
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

type JWTAuthConfig struct {
	Secret         string        `json:"secret"`
	ExpireDuration time.Duration `json:"expireDuration"`
	Issuer         string        `json:"issuer"`
	SigningMethod  string        `json:"signingMethod"`
	Realm          string        `json:"realm"` // JWT领域标识
}

type RateLimitConfig struct {
	Rate     int           `json:"rate"`
	Interval time.Duration `json:"interval"`
}

type MiddlewareConfig struct {
	Security  SecurityConfig  `json:"security"`
	JWT       JWTAuthConfig   `json:"jwt"`
	Timeout   TimeoutConfig   `json:"timeout"`
	CORS      CORSConfig      `json:"cors"`
	RateLimit RateLimitConfig `json:"rateLimit"`
}

// 新增数据库配置类型
type DatabaseConfig struct {
	Host        string `json:"host"`        // 数据库主机地址
	Port        int    `json:"port"`        // 数据库端口
	Username    string `json:"username"`    // 数据库用户名
	Password    string `json:"password"`    // 数据库密码
	DBName      string `json:"dbname"`      // 数据库名称
	UseUnixSock bool   `json:"useUnixSock"` // 是否使用Unix套接字连接
	MinPoolSize int    `json:"minPoolSize"` // 连接池最小连接数
	MaxPoolSize int    `json:"maxPoolSize"` // 连接池最大连接数
	LogLevel    string `json:"logLevel"`    // GORM日志级别
}

type Config struct {
	Server     ServerConfig     `json:"server"`
	Database   DatabaseConfig   `json:"database"` // 新增数据库配置节点
	Middleware MiddlewareConfig `json:"middleware"`
	Env        string           `json:"env"` // 环境标识
}

var defaultConfig = Config{
	Server: ServerConfig{
		Address: ":8080",
	},
	Database: DatabaseConfig{
		Host:        "localhost",
		Port:        3306,
		Username:    "root",
		Password:    "root",
		DBName:      "app",
		UseUnixSock: false,
		MinPoolSize: 5,
		MaxPoolSize: 50,
		LogLevel:    "warn",
	},
	Middleware: MiddlewareConfig{
		Security: SecurityConfig{
			MaxBodySize:    10 << 20, // 10MB
			AllowedMethods: []string{"GET", "POST", "PUT"},
		},
		JWT: JWTAuthConfig{ // JWT默认配置
			Secret:         "dev-secret-change-me-in-production", // 开发环境默认密钥
			ExpireDuration: 24 * time.Hour,
			Issuer:         "my-digital-home",
			SigningMethod:  "HS256",
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

	/****** JWT 配置 (新增部分) ******/
	if v := os.Getenv("JWT_SECRET"); v != "" {
		config.Middleware.JWT.Secret = v
	}

	if v := os.Getenv("JWT_EXPIRATION"); v != "" {
		if duration, err := time.ParseDuration(v); err == nil {
			config.Middleware.JWT.ExpireDuration = duration
		} else {
			hlog.Warnf("Invalid JWT_EXPIRATION format: %v", err)
		}
	}

	if v := os.Getenv("JWT_ISSUER"); v != "" {
		config.Middleware.JWT.Issuer = v
	}

	if v := os.Getenv("JWT_ALGORITHM"); v != "" {
		// 清理输入算法字符串中的空格
		algorithm := strings.ReplaceAll(v, " ", "")
		algorithm = strings.ToLower(algorithm)

		// 允许的算法列表
		validAlgorithms := map[string]bool{
			"hs256": true,
			"hs384": true,
			"hs512": true,
		}

		if validAlgorithms[algorithm] {
			// 统一转换为大写（标准JWT算法应全大写）
			config.Middleware.JWT.SigningMethod = strings.ToUpper(algorithm)
		} else {
			hlog.Warnf("Unsupported JWT algorithm: %s", v)
		}
	}
	// 数据库配置
	if v := os.Getenv("DB_HOST"); v != "" {
		config.Database.Host = v
	}

	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			config.Database.Port = port
		}
	}

	if v := os.Getenv("DB_USER"); v != "" {
		config.Database.Username = v
	}

	if v := os.Getenv("DB_PASSWORD"); v != "" {
		config.Database.Password = v
	}

	if v := os.Getenv("DB_NAME"); v != "" {
		config.Database.DBName = v
	}

	if v := os.Getenv("DB_SOCKET"); v != "" {
		config.Database.UseUnixSock = parseBool(v)
	}

	if v := os.Getenv("DB_MIN_POOL"); v != "" {
		if size, err := strconv.Atoi(v); err == nil {
			config.Database.MinPoolSize = size
		}
	}

	if v := os.Getenv("DB_MAX_POOL"); v != "" {
		if size, err := strconv.Atoi(v); err == nil {
			config.Database.MaxPoolSize = size
		}
	}

	if v := os.Getenv("DB_LOG_LEVEL"); v != "" {
		config.Database.LogLevel = strings.ToLower(v)
	}
}

// 分割环境变量列表（支持逗号分隔的字符串）
func splitEnvList(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}

// 转换字符串为布尔值
func parseBool(value string) bool {
	value = strings.ToLower(value)
	return value == "true" || value == "1" || value == "yes"
}

func (c *Config) InitDB() (*gorm.DB, error) {
	var dsn string
	charsetParam := "charset=utf8mb4&parseTime=True&loc=Local"

	// 自动切换连接方式
	if c.Database.UseUnixSock {
		dsn = fmt.Sprintf("%s:%s@unix(%s)/%s?%s",
			c.Database.Username,
			c.Database.Password,
			c.Database.Host, // 这里host存储的是socket路径
			c.Database.DBName,
			charsetParam)
	} else {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
			c.Database.Username,
			c.Database.Password,
			c.Database.Host,
			c.Database.Port,
			c.Database.DBName,
			charsetParam)
	}

	// 配置GORM日志级别
	gormConfig := &gorm.Config{}
	switch c.Database.LogLevel {
	case "silent":
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	case "error":
		gormConfig.Logger = logger.Default.LogMode(logger.Error)
	case "warn":
		gormConfig.Logger = logger.Default.LogMode(logger.Warn)
	case "info":
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}

	// 初始化数据库连接
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(c.Database.MinPoolSize)
	sqlDB.SetMaxOpenConns(c.Database.MaxPoolSize)

	return db, nil
}
