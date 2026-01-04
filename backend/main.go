package main

import (
	"log"
	"time"

	"blog/internal/config"
	"blog/internal/server"
)

func main() {
	// 设置全局默认时区为中国时区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Printf("加载中国时区失败: %v, 尝试使用UTC+8", err)
		loc = time.FixedZone("CST", 8*60*60)
	}
	time.Local = loc
	log.Printf("系统时区已设置为: %s, 当前时间: %s",
		time.Local.String(), time.Now().In(time.Local).Format("2006-01-02 15:04:05"))

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("加载配置失败:", err)
	}

	// 创建服务器
	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatal("创建服务器失败:", err)
	}
	defer srv.Close()

	// 启动服务器
	if err := srv.Start(); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
