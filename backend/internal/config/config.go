package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
			DBName: getEnv("DB_NAME", getEnv("POSTGRES_DB", "blog_db")),
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
	if cfg.Database.Password == "your_secure_password_here" {
		return nil, fmt.Errorf("数据库密码不能使用 .env.example 中的示例值")
	}

	adminUser := strings.TrimSpace(os.Getenv("ADMIN_USER"))
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	jwtSecret := os.Getenv("JWT_SECRET")
	if adminUser == "" {
		return nil, fmt.Errorf("管理员用户名未设置: 请设置 ADMIN_USER")
	}
	if len(adminPassword) < 12 || adminPassword == "admin123" || adminPassword == "replace_with_a_strong_password" {
		return nil, fmt.Errorf("管理员密码不安全: ADMIN_PASSWORD 至少需要 12 个字符且不能使用默认密码")
	}
	if len(jwtSecret) < 32 || jwtSecret == "your-secret-key-should-be-complex" || jwtSecret == "replace_with_at_least_32_random_characters" {
		return nil, fmt.Errorf("JWT_SECRET 至少需要 32 个字符且不能使用示例值")
	}
	if value := os.Getenv("COOKIE_SECURE"); value != "" {
		if _, err := strconv.ParseBool(value); err != nil {
			return nil, fmt.Errorf("COOKIE_SECURE 必须为 true 或 false")
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
