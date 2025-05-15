package tracking

import (
	"log"
	"net/http"
	"strconv"
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

	// 获取平均停留时间
	err = ts.db.QueryRow(`
		SELECT COALESCE(AVG(event_duration), 0) 
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

	rows, err := ts.db.Query(`
		SELECT page_path, COUNT(*) as visits
		FROM track_event
		WHERE event_type = 'PAGEVIEW'
		AND created_at BETWEEN $1 AND $2
		GROUP BY page_path
		ORDER BY visits DESC
		LIMIT 10
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var page string
		var visits int
		if err := rows.Scan(&page, &visits); err != nil {
			return nil, err
		}
		topPages.Pages = append(topPages.Pages, page)
		topPages.Visits = append(topPages.Visits, visits)
	}

	return topPages, nil
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

	// 获取所有页面作为节点
	rows, err := ts.db.Query(`
		SELECT page_path, COUNT(*) as visits
		FROM track_event
		WHERE event_type = 'PAGEVIEW'
		AND created_at BETWEEN $1 AND $2
		GROUP BY page_path
		ORDER BY visits DESC
		LIMIT 10
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pageMap := make(map[string]bool)
	for rows.Next() {
		var page string
		var visits int
		if err := rows.Scan(&page, &visits); err != nil {
			return nil, err
		}
		userPath.Nodes = append(userPath.Nodes, ChartData{Name: page, Value: visits})
		pageMap[page] = true
	}

	// 获取页面之间的转换关系
	rows, err = ts.db.Query(`
		WITH page_sequences AS (
			SELECT 
				session_id,
				page_path,
				LEAD(page_path) OVER (PARTITION BY session_id ORDER BY created_at) as next_page
			FROM track_event
			WHERE event_type = 'PAGEVIEW'
			AND created_at BETWEEN $1 AND $2
		)
		SELECT 
			page_path as source,
			next_page as target,
			COUNT(*) as value
		FROM page_sequences
		WHERE next_page IS NOT NULL
		GROUP BY source, target
		HAVING COUNT(*) >= 5
		ORDER BY value DESC
		LIMIT 20
	`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var link UserPathLink
		if err := rows.Scan(&link.Source, &link.Target, &link.Value); err != nil {
			return nil, err
		}
		// 只添加在节点列表中的页面之间的连接
		if pageMap[link.Source] && pageMap[link.Target] {
			userPath.Links = append(userPath.Links, link)
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
					WHEN user_agent ILIKE '%Chrome%' THEN 'Chrome'
					WHEN user_agent ILIKE '%Firefox%' THEN 'Firefox'
					WHEN user_agent ILIKE '%Safari%' THEN 'Safari'
					WHEN user_agent ILIKE '%Edge%' THEN 'Edge'
					WHEN user_agent ILIKE '%MSIE%' OR user_agent ILIKE '%Trident%' THEN 'IE'
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
