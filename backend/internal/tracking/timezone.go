package tracking

import (
	"log"
	"time"
)

var chinaLocation *time.Location

func init() {
	var err error
	chinaLocation, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Printf("加载中国时区失败: %v, 使用固定 UTC+8", err)
		chinaLocation = time.FixedZone("CST", 8*60*60)
	}
}
