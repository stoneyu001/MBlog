package tracking

import (
	"fmt"

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
	Overview OverviewStats   `json:"overview"`
	Trend    []DailyStats    `json:"trend"`
	TopPages []PageStats     `json:"top_pages"`
	Devices  []CategoryStats `json:"devices"`
	Browsers []CategoryStats `json:"browsers"`
	OS       []CategoryStats `json:"os"`
}

type OverviewStats struct {
	TotalPV            int64 `json:"total_pv"`
	TotalUV            int64 `json:"total_uv"`
	TodayPV            int64 `json:"today_pv"`
	TodayUV            int64 `json:"today_uv"`
	YesterdayPV        int64 `json:"yesterday_pv"`
	YesterdayUV        int64 `json:"yesterday_uv"`
	OnlineUsers        int64 `json:"online_users"`
	AverageReadSeconds int64 `json:"average_read_seconds"`
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
	resp := &StatsResponse{
		Trend:    []DailyStats{},
		TopPages: []PageStats{},
		Devices:  []CategoryStats{},
		Browsers: []CategoryStats{},
		OS:       []CategoryStats{},
	}

	overview, err := s.repo.OverviewStats()
	if err != nil {
		return nil, fmt.Errorf("获取概览数据: %w", err)
	}
	resp.Overview = OverviewStats{
		TotalPV:            overview.TotalPV,
		TotalUV:            overview.TotalUV,
		TodayPV:            overview.TodayPV,
		TodayUV:            overview.TodayUV,
		YesterdayPV:        overview.YesterdayPV,
		YesterdayUV:        overview.YesterdayUV,
		OnlineUsers:        overview.OnlineUsers,
		AverageReadSeconds: overview.AverageReadSeconds,
	}

	dailyStats, err := s.repo.DailyStats(7)
	if err != nil {
		return nil, fmt.Errorf("获取趋势数据: %w", err)
	}
	for _, d := range dailyStats {
		resp.Trend = append(resp.Trend, DailyStats{Date: d.Date, PV: d.PV, UV: d.UV})
	}

	topPages, err := s.repo.TopPages(10)
	if err != nil {
		return nil, fmt.Errorf("获取热门页面: %w", err)
	}
	for _, p := range topPages {
		resp.TopPages = append(resp.TopPages, PageStats{Path: p.Path, Title: p.Title, PV: p.PV, UV: p.UV})
	}

	// 注意: track_event.platform 列存储的是操作系统名称 (Windows/macOS/Linux 等)
	deviceStats, err := s.repo.CategoryStats("device_type")
	if err != nil {
		return nil, fmt.Errorf("获取设备统计: %w", err)
	}
	for _, c := range toCategoryStats(deviceStats) {
		resp.Devices = append(resp.Devices, c)
	}
	osStats, err := s.repo.CategoryStats("platform")
	if err != nil {
		return nil, fmt.Errorf("获取系统统计: %w", err)
	}
	for _, c := range toCategoryStats(osStats) {
		resp.OS = append(resp.OS, c)
	}
	browserStats, err := s.repo.MetadataStats("browser")
	if err != nil {
		return nil, fmt.Errorf("获取浏览器统计: %w", err)
	}
	for _, c := range toCategoryStats(browserStats) {
		resp.Browsers = append(resp.Browsers, c)
	}

	return resp, nil
}

func toCategoryStats(stats []repository.CategoryStat) []CategoryStats {
	result := make([]CategoryStats, len(stats))
	for i, s := range stats {
		result[i] = CategoryStats{Name: s.Name, Value: s.Value}
	}
	return result
}
