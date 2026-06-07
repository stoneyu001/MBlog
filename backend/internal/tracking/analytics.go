package tracking

import (
	"log"

	"blog/internal/repository"
)

// AnalyticsService 统计分析服务
type AnalyticsService struct {
	repo *repository.TrackingRepo
}

// NewAnalyticsService 创建统计分析服务
func NewAnalyticsService(repo *repository.TrackingRepo) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// StatsResponse 统计数据响应
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
	OnlineUsers int64 `json:"online_users"`
}

type DailyStats struct {
	Date string `json:"date"`
	PV   int64  `json:"pv"`
	UV   int64  `json:"uv"`
}

type PageStats struct {
	Path  string `json:"path"`
	Title string `json:"title"`
	PV    int64  `json:"pv"`
	UV    int64  `json:"uv"`
}

type CategoryStats struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// GetFullStats 获取完整统计数据
func (s *AnalyticsService) GetFullStats() (*StatsResponse, error) {
	resp := &StatsResponse{}

	overview, err := s.repo.OverviewStats()
	if err != nil {
		log.Printf("获取概览数据失败: %v", err)
	}
	resp.Overview = OverviewStats{
		TotalPV:     overview.TotalPV,
		TotalUV:     overview.TotalUV,
		TodayPV:     overview.TodayPV,
		TodayUV:     overview.TodayUV,
		YesterdayPV: overview.YesterdayPV,
		YesterdayUV: overview.YesterdayUV,
		OnlineUsers: overview.OnlineUsers,
	}

	dailyStats, err := s.repo.DailyStats(7)
	if err != nil {
		log.Printf("获取趋势数据失败: %v", err)
	}
	for _, d := range dailyStats {
		resp.Trend = append(resp.Trend, DailyStats{Date: d.Date, PV: d.PV, UV: d.UV})
	}

	topPages, err := s.repo.TopPages(10)
	if err != nil {
		log.Printf("获取热门页面失败: %v", err)
	}
	for _, p := range topPages {
		resp.TopPages = append(resp.TopPages, PageStats{Path: p.Path, Title: p.Title, PV: p.PV, UV: p.UV})
	}

	// 注意: track_event.platform 列存储的是操作系统名称 (Windows/macOS/Linux 等)
	for _, c := range toCategoryStats(s.repo.CategoryStats("device_type")) {
		resp.Devices = append(resp.Devices, c)
	}
	for _, c := range toCategoryStats(s.repo.CategoryStats("platform")) {
		resp.OS = append(resp.OS, c)
	}
	for _, c := range toCategoryStats(s.repo.MetadataStats("browser")) {
		resp.Browsers = append(resp.Browsers, c)
	}

	return resp, nil
}

func toCategoryStats(stats []repository.CategoryStat, err error) []CategoryStats {
	if err != nil {
		return nil
	}
	result := make([]CategoryStats, len(stats))
	for i, s := range stats {
		result[i] = CategoryStats{Name: s.Name, Value: s.Value}
	}
	return result
}
