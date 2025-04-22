package tracking

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 分析服务
type AnalyticsService struct {
	db *sql.DB
}

// 创建分析服务
func NewAnalyticsService(db *sql.DB) *AnalyticsService {
	return &AnalyticsService{db: db}
}

// 统计概览数据
type OverviewStats struct {
	TotalEvents    int64            `json:"total_events"`    // 总事件数
	PageViews      int64            `json:"page_views"`      // 页面浏览量
	UniqueVisitors int64            `json:"unique_visitors"` // 访客数
	ClickEvents    int64            `json:"click_events"`    // 点击事件数
	TopPages       []TopPageStat    `json:"top_pages"`       // 热门页面
	TopElements    []TopElementStat `json:"top_elements"`    // 热门元素
	EventsByHour   []HourStat       `json:"events_by_hour"`  // 时段统计
}

// 热门页面统计
type TopPageStat struct {
	PagePath string `json:"page_path"`
	Count    int64  `json:"count"`
}

// 热门元素统计
type TopElementStat struct {
	ElementPath string `json:"element_path"`
	Count       int64  `json:"count"`
}

// 时段统计
type HourStat struct {
	Hour  int   `json:"hour"`
	Count int64 `json:"count"`
}

// 注册分析接口处理程序
func (as *AnalyticsService) RegisterHandlers(router *gin.RouterGroup) {
	analytics := router.Group("/analytics")
	{
		// 概览统计
		analytics.GET("/overview", as.handleOverview)

		// 页面访问明细
		analytics.GET("/pageviews", as.handlePageViews)

		// 点击事件明细
		analytics.GET("/clicks", as.handleClickEvents)

		// 自定义事件明细
		analytics.GET("/custom", as.handleCustomEvents)
	}
}

// 获取概览统计
func (as *AnalyticsService) handleOverview(c *gin.Context) {
	// 获取时间范围
	startDate, endDate := getTimeRange(c)

	// 获取统计数据
	stats, err := as.getOverviewStats(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计数据失败"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// 获取页面访问统计
func (as *AnalyticsService) handlePageViews(c *gin.Context) {
	// 获取时间范围
	startDate, endDate := getTimeRange(c)

	// 分页参数
	page := getIntParam(c, "page", 1)
	pageSize := getIntParam(c, "page_size", 20)

	// 获取数据
	query := `
	SELECT page_path, COUNT(*) as count
	FROM track_events
	WHERE event_type = 'PAGEVIEW'
	AND created_at BETWEEN $1 AND $2
	GROUP BY page_path
	ORDER BY count DESC
	LIMIT $3 OFFSET $4
	`

	rows, err := as.db.Query(query, startDate, endDate, pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询数据失败"})
		return
	}
	defer rows.Close()

	var results []TopPageStat
	for rows.Next() {
		var stat TopPageStat
		if err := rows.Scan(&stat.PagePath, &stat.Count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "处理数据失败"})
			return
		}
		results = append(results, stat)
	}

	// 获取总数
	var total int64
	err = as.db.QueryRow(`
		SELECT COUNT(DISTINCT page_path) FROM track_events 
		WHERE event_type = 'PAGEVIEW' AND created_at BETWEEN $1 AND $2
	`, startDate, endDate).Scan(&total)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "统计总数失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       results,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	})
}

// 获取点击事件统计
func (as *AnalyticsService) handleClickEvents(c *gin.Context) {
	// 获取时间范围
	startDate, endDate := getTimeRange(c)

	// 分页参数
	page := getIntParam(c, "page", 1)
	pageSize := getIntParam(c, "page_size", 20)

	// 获取数据
	query := `
	SELECT element_path, page_path, COUNT(*) as count
	FROM track_events
	WHERE event_type = 'CLICK'
	AND created_at BETWEEN $1 AND $2
	GROUP BY element_path, page_path
	ORDER BY count DESC
	LIMIT $3 OFFSET $4
	`

	rows, err := as.db.Query(query, startDate, endDate, pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询数据失败"})
		return
	}
	defer rows.Close()

	type ClickStat struct {
		ElementPath string `json:"element_path"`
		PagePath    string `json:"page_path"`
		Count       int64  `json:"count"`
	}

	var results []ClickStat
	for rows.Next() {
		var stat ClickStat
		if err := rows.Scan(&stat.ElementPath, &stat.PagePath, &stat.Count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "处理数据失败"})
			return
		}
		results = append(results, stat)
	}

	// 获取总数
	var total int64
	err = as.db.QueryRow(`
		SELECT COUNT(DISTINCT element_path) FROM track_events 
		WHERE event_type = 'CLICK' AND created_at BETWEEN $1 AND $2
	`, startDate, endDate).Scan(&total)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "统计总数失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       results,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	})
}

// 获取自定义事件统计
func (as *AnalyticsService) handleCustomEvents(c *gin.Context) {
	// 获取时间范围
	startDate, endDate := getTimeRange(c)

	// 分页参数
	page := getIntParam(c, "page", 1)
	pageSize := getIntParam(c, "page_size", 20)

	// 获取数据
	query := `
	SELECT metadata, COUNT(*) as count
	FROM track_events
	WHERE event_type = 'CUSTOM'
	AND created_at BETWEEN $1 AND $2
	GROUP BY metadata
	ORDER BY count DESC
	LIMIT $3 OFFSET $4
	`

	rows, err := as.db.Query(query, startDate, endDate, pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询数据失败"})
		return
	}
	defer rows.Close()

	type CustomEventStat struct {
		Metadata json.RawMessage `json:"metadata"`
		Count    int64           `json:"count"`
	}

	var results []CustomEventStat
	for rows.Next() {
		var stat CustomEventStat
		if err := rows.Scan(&stat.Metadata, &stat.Count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "处理数据失败"})
			return
		}
		results = append(results, stat)
	}

	// 获取总数
	var total int64
	err = as.db.QueryRow(`
		SELECT COUNT(*) FROM track_events 
		WHERE event_type = 'CUSTOM' AND created_at BETWEEN $1 AND $2
	`, startDate, endDate).Scan(&total)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "统计总数失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       results,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	})
}

// 获取概览统计数据
func (as *AnalyticsService) getOverviewStats(startDate, endDate time.Time) (*OverviewStats, error) {
	stats := &OverviewStats{}

	// 总事件数
	err := as.db.QueryRow(`
		SELECT COUNT(*) FROM track_events 
		WHERE created_at BETWEEN $1 AND $2
	`, startDate, endDate).Scan(&stats.TotalEvents)
	if err != nil {
		return nil, err
	}

	// 页面浏览量
	err = as.db.QueryRow(`
		SELECT COUNT(*) FROM track_events 
		WHERE event_type = 'PAGEVIEW' AND created_at BETWEEN $1 AND $2
	`, startDate, endDate).Scan(&stats.PageViews)
	if err != nil {
		return nil, err
	}

	// 访客数
	err = as.db.QueryRow(`
		SELECT COUNT(DISTINCT session_id) FROM track_events 
		WHERE created_at BETWEEN $1 AND $2
	`, startDate, endDate).Scan(&stats.UniqueVisitors)
	if err != nil {
		return nil, err
	}

	// 点击事件数
	err = as.db.QueryRow(`
		SELECT COUNT(*) FROM track_events 
		WHERE event_type = 'CLICK' AND created_at BETWEEN $1 AND $2
	`, startDate, endDate).Scan(&stats.ClickEvents)
	if err != nil {
		return nil, err
	}

	// 热门页面
	rows, err := as.db.Query(`
		SELECT page_path, COUNT(*) as count
		FROM track_events
		WHERE event_type = 'PAGEVIEW' AND created_at BETWEEN $1 AND $2
		GROUP BY page_path
		ORDER BY count DESC
		LIMIT 5
	`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stat TopPageStat
		if err := rows.Scan(&stat.PagePath, &stat.Count); err != nil {
			return nil, err
		}
		stats.TopPages = append(stats.TopPages, stat)
	}

	// 热门元素
	rows, err = as.db.Query(`
		SELECT element_path, COUNT(*) as count
		FROM track_events
		WHERE event_type = 'CLICK' AND created_at BETWEEN $1 AND $2
		GROUP BY element_path
		ORDER BY count DESC
		LIMIT 5
	`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stat TopElementStat
		if err := rows.Scan(&stat.ElementPath, &stat.Count); err != nil {
			return nil, err
		}
		stats.TopElements = append(stats.TopElements, stat)
	}

	// 按小时统计
	rows, err = as.db.Query(`
		SELECT EXTRACT(HOUR FROM created_at) as hour, COUNT(*) as count
		FROM track_events
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY hour
		ORDER BY hour
	`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stat HourStat
		if err := rows.Scan(&stat.Hour, &stat.Count); err != nil {
			return nil, err
		}
		stats.EventsByHour = append(stats.EventsByHour, stat)
	}

	return stats, nil
}

// 获取时间范围参数
func getTimeRange(c *gin.Context) (time.Time, time.Time) {
	// 获取请求参数
	startStr := c.DefaultQuery("start_date", "")
	endStr := c.DefaultQuery("end_date", "")

	// 默认为近7天
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	// 解析自定义日期
	if startStr != "" {
		if parsedTime, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = parsedTime
		}
	}

	if endStr != "" {
		if parsedTime, err := time.Parse("2006-01-02", endStr); err == nil {
			// 设置为当天结束时间
			endDate = time.Date(parsedTime.Year(), parsedTime.Month(), parsedTime.Day(), 23, 59, 59, 999999999, parsedTime.Location())
		}
	} else {
		// 默认结束时间为今天结束
		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())
	}

	return startDate, endDate
}

// 获取整型参数
func getIntParam(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.DefaultQuery(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value := 0
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil || value <= 0 {
		return defaultValue
	}

	return value
}
