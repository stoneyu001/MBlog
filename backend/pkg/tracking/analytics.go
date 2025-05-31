package tracking

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AnalyticsMetrics 表示核心指标数据
type AnalyticsMetrics struct {
	TotalVisits    int     `json:"totalVisits"`
	UniqueVisitors int     `json:"uniqueVisitors"`
	AvgDuration    float64 `json:"avgDuration"`
	BounceRate     float64 `json:"bounceRate"`
}

// VisitsTrend 表示访问趋势数据
type VisitsTrend struct {
	Dates    []string `json:"dates"`
	Visits   []int    `json:"visits"`
	Visitors []int    `json:"visitors"`
}

// TopPages 表示热门页面数据
type TopPages struct {
	Pages  []string `json:"pages"`
	Visits []int    `json:"visits"`
}

// ChartData 表示图表数据项
type ChartData struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// UserPath 表示用户路径数据
type UserPath struct {
	Nodes []ChartData    `json:"nodes"`
	Links []UserPathLink `json:"links"`
}

// UserPathLink 表示用户路径连接
type UserPathLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Value  int    `json:"value"`
}

// AnalyticsResponse 表示分析数据响应
type AnalyticsResponse struct {
	Metrics       AnalyticsMetrics `json:"metrics"`
	VisitsTrend   VisitsTrend      `json:"visitsTrend"`
	TopPages      TopPages         `json:"topPages"`
	VisitDuration []ChartData      `json:"visitDuration"`
	EventTypes    []ChartData      `json:"eventTypes"`
	UserPaths     UserPath         `json:"userPaths"`
	Platforms     []ChartData      `json:"platforms"`
	Browsers      []ChartData      `json:"browsers"`
}

// HandleAnalytics 处理分析数据请求
func (ts *TrackingService) HandleAnalytics(c *gin.Context) {
	// 获取时间范围参数
	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil || days <= 0 {
		days = 30
	}

	// 计算时间范围
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	// 获取分析数据
	response, err := ts.getAnalyticsData(startTime, endTime)
	if err != nil {
		log.Printf("获取分析数据失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取数据失败"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// getAnalyticsData 获取分析数据
func (ts *TrackingService) getAnalyticsData(startTime, endTime time.Time) (*AnalyticsResponse, error) {
	response := &AnalyticsResponse{}

	// 获取核心指标
	metrics, err := ts.getMetrics(startTime, endTime)
	if err != nil {
		return nil, err
	}
	response.Metrics = *metrics

	// 获取访问趋势
	visitsTrend, err := ts.getVisitsTrend(startTime, endTime)
	if err != nil {
		return nil, err
	}
	response.VisitsTrend = *visitsTrend

	// 获取热门页面
	topPages, err := ts.getTopPages(startTime, endTime)
	if err != nil {
		return nil, err
	}
	response.TopPages = *topPages

	// 获取访问时长分布
	visitDuration, err := ts.getVisitDuration(startTime, endTime)
	if err != nil {
		return nil, err
	}
	response.VisitDuration = visitDuration

	// 获取事件类型分布
	eventTypes, err := ts.getEventTypes(startTime, endTime)
	if err != nil {
		return nil, err
	}
	response.EventTypes = eventTypes

	// 获取用户路径
	userPaths, err := ts.getUserPaths(startTime, endTime)
	if err != nil {
		return nil, err
	}
	response.UserPaths = *userPaths

	// 获取平台分布
	platforms, err := ts.getPlatforms(startTime, endTime)
	if err != nil {
		return nil, err
	}
	response.Platforms = platforms

	// 获取浏览器分布
	browsers, err := ts.getBrowsers(startTime, endTime)
	if err != nil {
		return nil, err
	}
	response.Browsers = browsers

	return response, nil
}

// getMetrics 获取核心指标数据
func (ts *TrackingService) getMetrics(startTime, endTime time.Time) (*AnalyticsMetrics, error) {
	metrics := &AnalyticsMetrics{}

	// 获取总访问量
	err := ts.db.QueryRow(`
		SELECT COUNT(*) 
		FROM track_event 
		WHERE event_type = 'PAGEVIEW' 
		AND created_at BETWEEN $1 AND $2
	`, startTime, endTime).Scan(&metrics.TotalVisits)
	if err != nil {
		return nil, err
	}

	// 获取独立访客数
	err = ts.db.QueryRow(`
		SELECT COUNT(DISTINCT user_id) 
		FROM track_event 
		WHERE created_at BETWEEN $1 AND $2
	`, startTime, endTime).Scan(&metrics.UniqueVisitors)
	if err != nil {
		return nil, err
	}

	// 获取平均停留时间（保留一位小数）
	err = ts.db.QueryRow(`
		SELECT ROUND(COALESCE(AVG(NULLIF(event_duration, 0)), 0)::numeric, 1)
		FROM track_event 
		WHERE event_duration > 0 
		AND created_at BETWEEN $1 AND $2
	`, startTime, endTime).Scan(&metrics.AvgDuration)
	if err != nil {
		return nil, err
	}

	// 计算跳出率（只访问一个页面的会话比例）
	var bounceCount, totalSessions int
	err = ts.db.QueryRow(`
		WITH session_pageviews AS (
			SELECT session_id, COUNT(*) as pageviews
			FROM track_event
			WHERE event_type = 'PAGEVIEW'
			AND created_at BETWEEN $1 AND $2
			GROUP BY session_id
		)
		SELECT 
			COUNT(CASE WHEN pageviews = 1 THEN 1 END) as bounce_count,
			COUNT(*) as total_sessions
		FROM session_pageviews
	`, startTime, endTime).Scan(&bounceCount, &totalSessions)
	if err != nil {
		return nil, err
	}

	if totalSessions > 0 {
		metrics.BounceRate = float64(bounceCount) / float64(totalSessions) * 100
	}

	return metrics, nil
}

// getVisitsTrend 获取访问趋势数据
func (ts *TrackingService) getVisitsTrend(startTime, endTime time.Time) (*VisitsTrend, error) {
	trend := &VisitsTrend{
		Dates:    make([]string, 0),
		Visits:   make([]int, 0),
		Visitors: make([]int, 0),
	}

	rows, err := ts.db.Query(`
		WITH dates AS (
			SELECT generate_series(
				date_trunc('day', $1::timestamp),
				date_trunc('day', $2::timestamp),
				'1 day'::interval
			) as date
		),
		daily_stats AS (
			SELECT 
				date_trunc('day', created_at) as day,
				COUNT(*) as visits,
				COUNT(DISTINCT user_id) as visitors
			FROM track_event
			WHERE event_type = 'PAGEVIEW'
			AND created_at BETWEEN $1 AND $2
			GROUP BY 1
		)
		SELECT 
			dates.date,
			COALESCE(daily_stats.visits, 0) as visits,
			COALESCE(daily_stats.visitors, 0) as visitors
		FROM dates
		LEFT JOIN daily_stats ON dates.date = daily_stats.day
		ORDER BY dates.date
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var date time.Time
		var visits, visitors int
		if err := rows.Scan(&date, &visits, &visitors); err != nil {
			return nil, err
		}
		trend.Dates = append(trend.Dates, date.Format("01-02"))
		trend.Visits = append(trend.Visits, visits)
		trend.Visitors = append(trend.Visitors, visitors)
	}

	return trend, nil
}

// getTopPages 获取热门页面数据
func (ts *TrackingService) getTopPages(startTime, endTime time.Time) (*TopPages, error) {
	topPages := &TopPages{
		Pages:  make([]string, 0),
		Visits: make([]int, 0),
	}

	// 添加调试日志
	log.Printf("查询热门页面，时间范围: %s 至 %s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))

	rows, err := ts.db.Query(`
		SELECT page_path, SUM(CASE 
			WHEN event_type = 'PAGEVIEW' THEN 1 
			WHEN event_type = 'CLICK' THEN 1 
			ELSE 0 
		END) as visits
		FROM track_event
		WHERE (event_type = 'PAGEVIEW' OR event_type = 'CLICK')
		AND page_path != '/'
		AND created_at BETWEEN $1 AND $2
		GROUP BY page_path
		ORDER BY visits DESC
		LIMIT 5
	`, startTime, endTime)
	if err != nil {
		log.Printf("热门页面查询错误: %v", err)
		return nil, err
	}
	defer rows.Close()

	// 添加计数器，确认是否有数据返回
	rowCount := 0

	for rows.Next() {
		var rawPath string
		var visits int
		if err := rows.Scan(&rawPath, &visits); err != nil {
			log.Printf("扫描热门页面结果错误: %v", err)
			return nil, err
		}

		rowCount++

		// 路径处理逻辑
		processedPath := processPagePath(rawPath)

		// 添加调试日志，记录原始路径和处理后的路径
		log.Printf("热门页面 #%d: 原始路径=%s, 处理后=%s, 访问量=%d",
			rowCount, rawPath, processedPath, visits)

		topPages.Pages = append(topPages.Pages, processedPath)
		topPages.Visits = append(topPages.Visits, visits)
	}

	// 检查是否有数据返回
	if rowCount == 0 {
		log.Printf("警告: 热门页面查询没有返回任何结果!")
	}

	return topPages, nil
}

// processPagePath 处理页面路径，提取更友好的显示名称
func processPagePath(rawPath string) string {
	// 处理根路径
	if rawPath == "/" {
		return "首页"
	}

	// 分割路径并取最后部分
	parts := strings.Split(rawPath, "/")
	if len(parts) == 0 {
		return "未知页面"
	}

	lastPart := parts[len(parts)-1]
	if lastPart == "" && len(parts) > 1 {
		lastPart = parts[len(parts)-2]
	}

	// 如果还是空，返回原始路径
	if lastPart == "" {
		return rawPath
	}

	// 移除文件扩展名
	if strings.Contains(lastPart, ".") {
		extParts := strings.Split(lastPart, ".")
		if len(extParts) > 1 {
			lastPart = strings.Join(extParts[:len(extParts)-1], ".")
		}
	}

	// URL 解码（处理中文字符）
	decoded, err := url.QueryUnescape(lastPart)
	if err == nil {
		lastPart = decoded
	}
	return lastPart
}

// getVisitDuration 获取访问时长分布
func (ts *TrackingService) getVisitDuration(startTime, endTime time.Time) ([]ChartData, error) {
	var result []ChartData

	rows, err := ts.db.Query(`
		WITH duration_ranges AS (
			SELECT 
				CASE 
					WHEN event_duration < 10 THEN '0-10秒'
					WHEN event_duration < 30 THEN '10-30秒'
					WHEN event_duration < 60 THEN '30-60秒'
					WHEN event_duration < 180 THEN '1-3分钟'
					WHEN event_duration < 300 THEN '3-5分钟'
					ELSE '5分钟以上'
				END as duration_range,
				COUNT(*) as count
			FROM track_event
			WHERE event_duration > 0
			AND created_at BETWEEN $1 AND $2
			GROUP BY 1
		)
		SELECT duration_range, count
		FROM duration_ranges
		ORDER BY count DESC
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result = append(result, ChartData{Name: name, Value: value})
	}

	return result, nil
}

// getEventTypes 获取事件类型分布
func (ts *TrackingService) getEventTypes(startTime, endTime time.Time) ([]ChartData, error) {
	var result []ChartData

	rows, err := ts.db.Query(`
		SELECT event_type, COUNT(*) as count
		FROM track_event
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY event_type
		ORDER BY count DESC
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result = append(result, ChartData{Name: name, Value: value})
	}

	return result, nil
}

// getUserPaths 获取用户路径数据
func (ts *TrackingService) getUserPaths(startTime, endTime time.Time) (*UserPath, error) {
	userPath := &UserPath{
		Nodes: make([]ChartData, 0),
		Links: make([]UserPathLink, 0),
	}

	// 获取所有页面和点击事件作为节点
	rows, err := ts.db.Query(`
		WITH event_counts AS (
			SELECT 
				event_type,
				CASE 
					WHEN event_type = 'CLICK' THEN element_path
					ELSE page_path
				END as path,
				COUNT(*) as visits
			FROM track_event
			WHERE (event_type = 'PAGEVIEW' OR event_type = 'CLICK')
			AND created_at BETWEEN $1 AND $2
			GROUP BY 1, 2
			HAVING COUNT(*) > 0
			ORDER BY visits DESC
			LIMIT 15
		)
		SELECT event_type, path, visits 
		FROM event_counts
		WHERE path IS NOT NULL AND path != ''
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 记录节点映射，用于后续处理
	nodeMap := make(map[string]bool)
	for rows.Next() {
		var eventType, path string
		var visits int
		if err := rows.Scan(&eventType, &path, &visits); err != nil {
			return nil, err
		}

		// 处理节点名称
		var nodeName string
		if eventType == "CLICK" {
			nodeName = "点击: " + path
			if path == "" {
				nodeName = "点击: 未知元素"
			}
		} else {
			nodeName = processPagePath(path)
		}

		// 处理显示名称
		displayName := nodeName
		if len(displayName) > 30 {
			displayName = displayName[:27] + "..."
		}

		userPath.Nodes = append(userPath.Nodes, ChartData{Name: displayName, Value: visits})
		nodeMap[nodeName] = true
	}

	// 如果节点太少，不生成路径
	if len(userPath.Nodes) < 2 {
		return userPath, nil
	}

	// 获取事件之间的转换关系
	rows, err = ts.db.Query(`
		WITH event_sequence AS (
			SELECT 
				session_id,
				event_type,
				CASE 
					WHEN event_type = 'CLICK' THEN element_path
					ELSE page_path
				END as path,
				created_at,
				LEAD(event_type) OVER (PARTITION BY session_id ORDER BY created_at) as next_event_type,
				LEAD(
					CASE 
						WHEN event_type = 'CLICK' THEN element_path
						ELSE page_path
					END
				) OVER (PARTITION BY session_id ORDER BY created_at) as next_path
			FROM track_event
			WHERE (event_type = 'PAGEVIEW' OR event_type = 'CLICK')
			AND created_at BETWEEN $1 AND $2
		)
		SELECT 
			event_type, path,
			next_event_type, next_path,
			COUNT(*) as value
		FROM event_sequence
		WHERE next_path IS NOT NULL
		AND (event_type != next_event_type OR path != next_path)
		GROUP BY 1, 2, 3, 4
		HAVING COUNT(*) > 0
		ORDER BY value DESC
		LIMIT 30
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var eventType, path, nextEventType, nextPath string
		var value int
		if err := rows.Scan(&eventType, &path, &nextEventType, &nextPath, &value); err != nil {
			return nil, err
		}

		// 处理源节点和目标节点名称
		var source, target string
		if eventType == "CLICK" {
			source = "点击: " + path
			if path == "" {
				source = "点击: 未知元素"
			}
		} else {
			source = processPagePath(path)
		}

		if nextEventType == "CLICK" {
			target = "点击: " + nextPath
			if nextPath == "" {
				target = "点击: 未知元素"
			}
		} else {
			target = processPagePath(nextPath)
		}

		// 只添加在节点列表中的连接
		if nodeMap[source] && nodeMap[target] {
			// 处理显示名称
			displaySource := source
			displayTarget := target
			if len(displaySource) > 30 {
				displaySource = displaySource[:27] + "..."
			}
			if len(displayTarget) > 30 {
				displayTarget = displayTarget[:27] + "..."
			}
			userPath.Links = append(userPath.Links, UserPathLink{
				Source: displaySource,
				Target: displayTarget,
				Value:  value,
			})
		}
	}

	return userPath, nil
}

// getPlatforms 获取平台分布
func (ts *TrackingService) getPlatforms(startTime, endTime time.Time) ([]ChartData, error) {
	var result []ChartData

	rows, err := ts.db.Query(`
		SELECT platform, COUNT(*) as count
		FROM track_event
		WHERE created_at BETWEEN $1 AND $2
		AND platform IS NOT NULL
		GROUP BY platform
		ORDER BY count DESC
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result = append(result, ChartData{Name: name, Value: value})
	}

	return result, nil
}

// getBrowsers 获取浏览器分布
func (ts *TrackingService) getBrowsers(startTime, endTime time.Time) ([]ChartData, error) {
	var result []ChartData

	rows, err := ts.db.Query(`
		WITH browser_info AS (
			SELECT 
				CASE 
					WHEN user_agent ILIKE '%edg/%' OR user_agent ILIKE '%edge/%' THEN 'Edge'
					WHEN user_agent ILIKE '%chrome%' AND NOT (user_agent ILIKE '%edg/%' OR user_agent ILIKE '%edge/%') THEN 'Chrome'
					WHEN user_agent ILIKE '%firefox%' THEN 'Firefox'
					WHEN user_agent ILIKE '%safari%' AND NOT user_agent ILIKE '%chrome%' THEN 'Safari'
					WHEN user_agent ILIKE '%opera%' OR user_agent ILIKE '%opr/%' THEN 'Opera'
					WHEN user_agent ILIKE '%msie%' OR user_agent ILIKE '%trident%' THEN 'IE'
					ELSE 'Other'
				END as browser,
				COUNT(*) as count
			FROM track_event
			WHERE created_at BETWEEN $1 AND $2
			GROUP BY 1
		)
		SELECT browser, count
		FROM browser_info
		ORDER BY count DESC
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result = append(result, ChartData{Name: name, Value: value})
	}

	return result, nil
}
