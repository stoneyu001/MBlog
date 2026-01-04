package config

import (
	"fmt"
	"os"
)

// Config 应用配置
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string
}

// LoadConfig 从环境变量加载配置
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:   getEnv("DB_HOST", "db"),
			Port:   getEnv("DB_PORT", "5432"),
			User:   getEnv("DB_USER", "postgres"),
			DBName: getEnv("POSTGRES_DB", "blog_db"),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "3000"),
		},
	}

	// 密码必须从环境变量获取，不提供默认值
	cfg.Database.Password = os.Getenv("DB_PASSWORD")
	if cfg.Database.Password == "" {
		cfg.Database.Password = os.Getenv("POSTGRES_PASSWORD")
		if cfg.Database.Password == "" {
			return nil, fmt.Errorf("数据库密码未设置: 请在环境变量中设置 DB_PASSWORD 或 POSTGRES_PASSWORD")
		}
	}

	return cfg, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
