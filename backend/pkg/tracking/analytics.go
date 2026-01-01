package tracking

import (
	"database/sql"
	"log"
)

// AnalyticsService 处理统计分析
type AnalyticsService struct {
	db *sql.DB
}

// NewAnalyticsService 创建新的统计服务
func NewAnalyticsService(db *sql.DB) *AnalyticsService {
	return &AnalyticsService{db: db}
}

// StatsResponse 统计数据响应结构
type StatsResponse struct {
	Overview  OverviewStats   `json:"overview"`
	Trend     []DailyStats    `json:"trend"`
	TopPages  []PageStats     `json:"top_pages"`
	Devices   []CategoryStats `json:"devices"`
	Browsers  []CategoryStats `json:"browsers"`
	OS        []CategoryStats `json:"os"`
	Locations []CategoryStats `json:"locations"`
}

type OverviewStats struct {
	TotalPV     int64 `json:"total_pv"`
	TotalUV     int64 `json:"total_uv"`
	TodayPV     int64 `json:"today_pv"`
	TodayUV     int64 `json:"today_uv"`
	YesterdayPV int64 `json:"yesterday_pv"`
	YesterdayUV int64 `json:"yesterday_uv"`
	OnlineUsers int64 `json:"online_users"` // 过去5分钟活跃
}

type DailyStats struct {
	Date string `json:"date"`
	PV   int64  `json:"pv"`
	UV   int64  `json:"uv"`
}

type PageStats struct {
	Path  string `json:"path"`
	Title string `json:"title"` // 预留，目前可能和Path一样
	PV    int64  `json:"pv"`
	UV    int64  `json:"uv"`
}

type CategoryStats struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// GetFullStats 获取所有统计数据
func (s *AnalyticsService) GetFullStats() (*StatsResponse, error) {
	resp := &StatsResponse{}
	var err error

	if resp.Overview, err = s.getOverviewStats(); err != nil {
		log.Printf("获取概览数据失败: %v", err)
	}

	if resp.Trend, err = s.getTrendStats(7); err != nil {
		log.Printf("获取趋势数据失败: %v", err)
	}

	if resp.TopPages, err = s.getTopPages(10); err != nil {
		log.Printf("获取热门页面失败: %v", err)
	}

	if resp.Devices, err = s.getCategoryStats("device_type"); err != nil {
		log.Printf("获取设备统计失败: %v", err)
	}

	// 从 user_agent 或 platform 字段统计
	if resp.OS, err = s.getCategoryStats("platform"); err != nil {
		log.Printf("获取系统统计失败: %v", err)
	}

	// 浏览器统计需要解析 user_agent，这里简化处理，假设 metadata 中有 browser 字段
	// 或者我们直接查询 metadata->>'browser'
	if resp.Browsers, err = s.getMetadataStats("browser"); err != nil {
		log.Printf("获取浏览器统计失败: %v", err)
	}

	return resp, nil
}

func (s *AnalyticsService) getOverviewStats() (OverviewStats, error) {
	var stats OverviewStats

	// 总计
	s.db.QueryRow("SELECT COUNT(*), COUNT(DISTINCT session_id) FROM track_event").Scan(&stats.TotalPV, &stats.TotalUV)

	// 今日 (使用数据库时间，注意时区)
	// Postgres 的 CURRENT_DATE 默认是基于服务器时区的
	todayQuery := `
		SELECT COUNT(*), COUNT(DISTINCT session_id) 
		FROM track_event 
		WHERE created_at >= CURRENT_DATE`
	s.db.QueryRow(todayQuery).Scan(&stats.TodayPV, &stats.TodayUV)

	// 昨日
	yesterdayQuery := `
		SELECT COUNT(*), COUNT(DISTINCT session_id) 
		FROM track_event 
		WHERE created_at >= CURRENT_DATE - INTERVAL '1 day' 
		AND created_at < CURRENT_DATE`
	s.db.QueryRow(yesterdayQuery).Scan(&stats.YesterdayPV, &stats.YesterdayUV)

	// 在线用户 (过去5分钟)
	onlineQuery := `
		SELECT COUNT(DISTINCT session_id) 
		FROM track_event 
		WHERE created_at >= NOW() - INTERVAL '5 minutes'`
	s.db.QueryRow(onlineQuery).Scan(&stats.OnlineUsers)

	return stats, nil
}

func (s *AnalyticsService) getTrendStats(days int) ([]DailyStats, error) {
	query := `
		SELECT 
			TO_CHAR(created_at, 'YYYY-MM-DD') as date,
			COUNT(*) as pv,
			COUNT(DISTINCT session_id) as uv
		FROM track_event
		WHERE created_at >= CURRENT_DATE - INTERVAL '7 days'
		GROUP BY date
		ORDER BY date ASC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DailyStats
	for rows.Next() {
		var d DailyStats
		if err := rows.Scan(&d.Date, &d.PV, &d.UV); err != nil {
			continue
		}
		results = append(results, d)
	}
	return results, nil
}

func (s *AnalyticsService) getTopPages(limit int) ([]PageStats, error) {
	// 排除静态资源和管理页面
	query := `
		SELECT 
			page_path, 
			COUNT(*) as pv,
			COUNT(DISTINCT session_id) as uv
		FROM track_event
		WHERE page_path NOT LIKE '/static/%' 
		  AND page_path NOT LIKE '/admin%'
		  AND page_path != ''
		GROUP BY page_path
		ORDER BY pv DESC
		LIMIT $1
	`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PageStats
	for rows.Next() {
		var p PageStats
		if err := rows.Scan(&p.Path, &p.PV, &p.UV); err != nil {
			continue
		}
		p.Title = p.Path // 暂时用路径作为标题
		results = append(results, p)
	}
	return results, nil
}

func (s *AnalyticsService) getCategoryStats(column string) ([]CategoryStats, error) {
	// 动态列名查询需要小心 SQL 注入，但这里是内部调用，column 是受控的
	query := "SELECT " + column + ", COUNT(*) as count FROM track_event WHERE " + column + " IS NOT NULL AND " + column + " != '' GROUP BY " + column + " ORDER BY count DESC LIMIT 10"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CategoryStats
	for rows.Next() {
		var c CategoryStats
		if err := rows.Scan(&c.Name, &c.Value); err != nil {
			continue
		}
		results = append(results, c)
	}
	return results, nil
}

func (s *AnalyticsService) getMetadataStats(key string) ([]CategoryStats, error) {
	// 查询 JSONB 字段
	query := `
		SELECT 
			metadata->>$1 as name, 
			COUNT(*) as count 
		FROM track_event 
		WHERE metadata->>$1 IS NOT NULL 
		  AND metadata->>$1 != '' 
		GROUP BY name 
		ORDER BY count DESC 
		LIMIT 10
	`

	rows, err := s.db.Query(query, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CategoryStats
	for rows.Next() {
		var c CategoryStats
		if err := rows.Scan(&c.Name, &c.Value); err != nil {
			continue
		}
		results = append(results, c)
	}
	return results, nil
}
