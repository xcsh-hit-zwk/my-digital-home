package config

import "os"

type ServerConfig struct {
	Address string
}

type Config struct {
	Server ServerConfig
	Env    string // 环境标识（development/production）
}

// IsProd 判断当前是否生产环境
func (c *Config) IsProd() bool {
	return c.Env == "production"
}

// Load 多源配置加载（环境变量+默认值）
func Load() *Config {
	env := "development" // 默认开发环境
	if v := os.Getenv("APP_ENV"); v != "" {
		env = v
	}

	return &Config{
		Env: env,
		Server: ServerConfig{
			Address: getEnvWithDefault("SERVER_ADDR", ":8080"),
		},
	}
}

// getEnvWithDefault 带默认值的环境变量读取
func getEnvWithDefault(key, defVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defVal
}
