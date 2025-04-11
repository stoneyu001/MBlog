package database

import "fmt"

var connStr = fmt.Sprintf("postgres://%s:%s@%s:5432/%s",
	"postgres", // 用户名
	"4341289",  // 密码
	"db",       // Docker服务名（容器间通信用）
	"blog_db")  // 数据库名
