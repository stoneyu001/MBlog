package tracking

import (
	"database/sql"
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
		// 不分区数据概览
		analytics.GET("/overview", as.handleUnpartitionedOverview)

		// 不分区数据明细
		analytics.GET("/details", as.handleUnpartitionedDetails)

		// 平台统计
		analytics.GET("/platforms", as.handlePlatformStats)
	}
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

// 不分区埋点数据概览
func (as *AnalyticsService) handleUnpartitionedOverview(c *gin.Context) {
	// 获取时间范围
	startDate, endDate := getTimeRange(c)

	// 平台筛选
	platform := c.Query("platform")

	// 构建基础查询
	query := `
	SELECT 
		COUNT(*) as total_events,
		COUNT(DISTINCT session_id) as unique_visitors,
		COUNT(DISTINCT user_id) as unique_users,
		COUNT(CASE WHEN event_type = 'PAGEVIEW' THEN 1 END) as page_views,
		COUNT(CASE WHEN event_type = 'CLICK' THEN 1 END) as click_events
	FROM track_events_unpartitioned
	WHERE created_at BETWEEN $1 AND $2
	`

	args := []interface{}{startDate, endDate}
	argIndex := 3

	// 添加平台筛选条件
	if platform != "" {
		query += fmt.Sprintf(" AND platform = $%d", argIndex)
		args = append(args, platform)
		argIndex++
	}

	// 查询总体概览数据
	var stats struct {
		TotalEvents    int64 `json:"total_events"`
		UniqueVisitors int64 `json:"unique_visitors"`
		UniqueUsers    int64 `json:"unique_users"`
		PageViews      int64 `json:"page_views"`
		ClickEvents    int64 `json:"click_events"`
	}

	err := as.db.QueryRow(query, args...).Scan(
		&stats.TotalEvents,
		&stats.UniqueVisitors,
		&stats.UniqueUsers,
		&stats.PageViews,
		&stats.ClickEvents,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询概览数据失败"})
		return
	}

	// 查询事件类型分布
	eventTypesQuery := `
	SELECT event_type, COUNT(*) as count
	FROM track_events_unpartitioned
	WHERE created_at BETWEEN $1 AND $2
	`

	if platform != "" {
		eventTypesQuery += fmt.Sprintf(" AND platform = $%d", argIndex)
	}

	eventTypesQuery += " GROUP BY event_type ORDER BY count DESC LIMIT 10"

	rows, err := as.db.Query(eventTypesQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询事件类型分布失败"})
		return
	}
	defer rows.Close()

	var eventTypes []struct {
		Type  string `json:"event_type"`
		Count int64  `json:"count"`
	}

	for rows.Next() {
		var et struct {
			Type  string `json:"event_type"`
			Count int64  `json:"count"`
		}
		if err := rows.Scan(&et.Type, &et.Count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "处理事件类型数据失败"})
			return
		}
		eventTypes = append(eventTypes, et)
	}

	// 查询热门页面
	pagesQuery := `
	SELECT page_path, COUNT(*) as count
	FROM track_events_unpartitioned
	WHERE event_type = 'PAGEVIEW' AND created_at BETWEEN $1 AND $2
	`

	if platform != "" {
		pagesQuery += fmt.Sprintf(" AND platform = $%d", argIndex)
	}

	pagesQuery += " GROUP BY page_path ORDER BY count DESC LIMIT 5"

	rows, err = as.db.Query(pagesQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询热门页面失败"})
		return
	}
	defer rows.Close()

	var topPages []struct {
		Path  string `json:"page_path"`
		Count int64  `json:"count"`
	}

	for rows.Next() {
		var page struct {
			Path  string `json:"page_path"`
			Count int64  `json:"count"`
		}
		if err := rows.Scan(&page.Path, &page.Count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "处理热门页面数据失败"})
			return
		}
		topPages = append(topPages, page)
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"stats":       stats,
		"event_types": eventTypes,
		"top_pages":   topPages,
		"start_date":  startDate.Format("2006-01-02"),
		"end_date":    endDate.Format("2006-01-02"),
		"platform":    platform,
	})
}

// 不分区埋点数据明细查询
func (as *AnalyticsService) handleUnpartitionedDetails(c *gin.Context) {
	// 获取时间范围
	startDate, endDate := getTimeRange(c)

	// 筛选条件
	platform := c.Query("platform")
	eventType := c.Query("event_type")
	appVersion := c.Query("app_version")

	// 分页参数
	page := getIntParam(c, "page", 1)
	pageSize := getIntParam(c, "page_size", 20)

	// 构建查询
	query := `
	SELECT 
		id, session_id, user_id, event_type, element_path, page_path, 
		referrer, platform, event_source, app_version, event_duration, created_at
	FROM track_events_unpartitioned
	WHERE created_at BETWEEN $1 AND $2
	`

	countQuery := `
	SELECT COUNT(*) FROM track_events_unpartitioned
	WHERE created_at BETWEEN $1 AND $2
	`

	args := []interface{}{startDate, endDate}
	argIndex := 3

	// 添加筛选条件
	if platform != "" {
		query += fmt.Sprintf(" AND platform = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND platform = $%d", argIndex)
		args = append(args, platform)
		argIndex++
	}

	if eventType != "" {
		query += fmt.Sprintf(" AND event_type = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND event_type = $%d", argIndex)
		args = append(args, eventType)
		argIndex++
	}

	if appVersion != "" {
		query += fmt.Sprintf(" AND app_version = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND app_version = $%d", argIndex)
		args = append(args, appVersion)
		argIndex++
	}

	// 添加排序和分页
	query += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", argIndex) + " OFFSET $" + fmt.Sprintf("%d", argIndex+1)
	args = append(args, pageSize, (page-1)*pageSize)

	// 执行查询
	rows, err := as.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询数据失败"})
		return
	}
	defer rows.Close()

	// 定义明细数据结构
	type EventDetail struct {
		ID            int64     `json:"id"`
		SessionID     string    `json:"session_id"`
		UserID        string    `json:"user_id"`
		EventType     string    `json:"event_type"`
		ElementPath   string    `json:"element_path"`
		PagePath      string    `json:"page_path"`
		Referrer      string    `json:"referrer"`
		Platform      string    `json:"platform"`
		EventSource   string    `json:"event_source"`
		AppVersion    string    `json:"app_version"`
		EventDuration int       `json:"event_duration"`
		CreatedAt     time.Time `json:"created_at"`
	}

	var events []EventDetail
	for rows.Next() {
		var e EventDetail
		if err := rows.Scan(
			&e.ID, &e.SessionID, &e.UserID, &e.EventType, &e.ElementPath, &e.PagePath,
			&e.Referrer, &e.Platform, &e.EventSource, &e.AppVersion, &e.EventDuration, &e.CreatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "处理数据失败"})
			return
		}
		events = append(events, e)
	}

	// 查询总数
	var total int64
	err = as.db.QueryRow(countQuery, args[:argIndex-1]...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "统计总数失败"})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"data":       events,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
		"filters": gin.H{
			"platform":    platform,
			"event_type":  eventType,
			"app_version": appVersion,
		},
	})
}

// 平台分布统计
func (as *AnalyticsService) handlePlatformStats(c *gin.Context) {
	// 获取时间范围
	startDate, endDate := getTimeRange(c)

	// 平台统计查询
	query := `
	SELECT 
		platform, 
		COUNT(*) as event_count,
		COUNT(DISTINCT session_id) as session_count,
		COUNT(DISTINCT user_id) as user_count,
		COUNT(CASE WHEN event_type = 'PAGEVIEW' THEN 1 END) as pageview_count
	FROM track_events_unpartitioned
	WHERE created_at BETWEEN $1 AND $2 AND platform IS NOT NULL
	GROUP BY platform
	ORDER BY event_count DESC
	`

	rows, err := as.db.Query(query, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询平台统计失败"})
		return
	}
	defer rows.Close()

	type PlatformStat struct {
		Platform      string `json:"platform"`
		EventCount    int64  `json:"event_count"`
		SessionCount  int64  `json:"session_count"`
		UserCount     int64  `json:"user_count"`
		PageviewCount int64  `json:"pageview_count"`
	}

	var stats []PlatformStat
	for rows.Next() {
		var s PlatformStat
		if err := rows.Scan(&s.Platform, &s.EventCount, &s.SessionCount, &s.UserCount, &s.PageviewCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "处理平台统计数据失败"})
			return
		}
		stats = append(stats, s)
	}

	// 应用版本统计查询
	versionQuery := `
	SELECT 
		platform,
		app_version, 
		COUNT(*) as count
	FROM track_events_unpartitioned
	WHERE created_at BETWEEN $1 AND $2 
	  AND platform IS NOT NULL 
	  AND app_version IS NOT NULL
	GROUP BY platform, app_version
	ORDER BY platform, count DESC
	`

	vrows, err := as.db.Query(versionQuery, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询版本统计失败"})
		return
	}
	defer vrows.Close()

	type VersionStat struct {
		Platform   string `json:"platform"`
		AppVersion string `json:"app_version"`
		Count      int64  `json:"count"`
	}

	var versions []VersionStat
	for vrows.Next() {
		var v VersionStat
		if err := vrows.Scan(&v.Platform, &v.AppVersion, &v.Count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "处理版本统计数据失败"})
			return
		}
		versions = append(versions, v)
	}

	c.JSON(http.StatusOK, gin.H{
		"platforms":  stats,
		"versions":   versions,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	})
}
