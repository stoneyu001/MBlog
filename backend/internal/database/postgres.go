package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"blog/internal/config"

	_ "github.com/lib/pq" // PostgreSQL 驱动
)

// NewPostgresDB 创建并配置 PostgreSQL 数据库连接
func NewPostgresDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	// 构建连接字符串
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&timezone=Asia/Shanghai&client_encoding=UTF8",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)

	log.Printf("正在连接数据库: %s@%s:%s/%s", cfg.User, cfg.Host, cfg.Port, cfg.DBName)

	// 打开数据库连接
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(25)                 // 最大打开连接数
	db.SetMaxIdleConns(5)                  // 最大空闲连接数
	db.SetConnMaxLifetime(5 * time.Minute) // 连接最大生命周期

	// 验证数据库连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接验证失败: %w", err)
	}

	log.Printf("数据库连接成功 - 连接池配置: 最大连接=%d, 空闲连接=%d, 生命周期=%v",
		25, 5, 5*time.Minute)

	return db, nil
}
