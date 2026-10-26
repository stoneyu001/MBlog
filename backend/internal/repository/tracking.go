package repository

import (
	"database/sql"
	"log"
	"net/url"
	"strings"
	"time"
)

// TrackingRepo 埋点数据访问层
type TrackingRepo struct {
	db *sql.DB
}

// NewTrackingRepo 创建埋点 Repository
func NewTrackingRepo(db *sql.DB) *TrackingRepo {
	return &TrackingRepo{db: db}
}

// === 写入操作 ===

// BatchInsertEvents 批量插入埋点事件（在事务中执行）
func (r *TrackingRepo) BatchInsertEvents(events []TrackEventRow) (success, fail int) {
	tx, err := r.db.Begin()
	if err != nil {
		log.Printf("事务启动失败: %v，回退到单条插入", err)
		for _, e := range events {
			if r.insertOne(e) == nil {
				success++
			} else {
				fail++
			}
		}
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(`
		INSERT INTO track_event
		(session_id, user_id, event_type, element_path, page_path, referrer,
		metadata, user_agent, ip_address, created_at, custom_properties,
		platform, device_info, event_duration, device_id, version, device_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11::jsonb,
		$12, $13::jsonb, $14, $15, $16, $17)
	`)
	if err != nil {
		log.Printf("准备语句失败: %v", err)
		for _, e := range events {
			if r.insertOne(e) == nil {
				success++
			} else {
				fail++
			}
		}
		return
	}
	defer stmt.Close()

	for _, e := range events {
		e.defaults()
		_, execErr := stmt.Exec(
			e.SessionID, e.UserID, e.EventType, e.ElementPath, e.PagePath,
			e.Referrer, e.Metadata, e.UserAgent, e.IPAddress, e.CreatedAt,
			e.CustomProperties, e.Platform, e.DeviceInfo, e.EventDuration,
			e.DeviceID, e.Version, e.DeviceType,
		)
		if execErr != nil {
			log.Printf("插入事件失败: %v", execErr)
			fail++
		} else {
			success++
		}
	}

	if err = tx.Commit(); err != nil {
		log.Printf("提交事务失败: %v", err)
	}
	return
}

// InsertOne 单条插入（队列满时的降级路径）
func (r *TrackingRepo) InsertOne(e TrackEventRow) error {
	return r.insertOne(e)
}

func (r *TrackingRepo) insertOne(e TrackEventRow) error {
	e.defaults()
	_, err := r.db.Exec(`
		INSERT INTO track_event
		(session_id, user_id, event_type, element_path, page_path, referrer,
		metadata, user_agent, ip_address, created_at, custom_properties,
		platform, device_info, event_duration, device_id, version, device_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11::jsonb,
		$12, $13::jsonb, $14, $15, $16, $17)
	`,
		e.SessionID, e.UserID, e.EventType, e.ElementPath, e.PagePath,
		e.Referrer, e.Metadata, e.UserAgent, e.IPAddress, e.CreatedAt,
		e.CustomProperties, e.Platform, e.DeviceInfo, e.EventDuration,
		e.DeviceID, e.Version, e.DeviceType,
	)
	if err != nil {
		log.Printf("单条插入失败: %v", err)
	}
	return err
}

// === 读取操作：概览 ===

// OverviewStats 概览统计数据
type OverviewStats struct {
	TotalPV, TotalUV, TodayPV, TodayUV, YesterdayPV, YesterdayUV, OnlineUsers, AverageReadSeconds int64
}

func (r *TrackingRepo) OverviewStats() (OverviewStats, error) {
	var s OverviewStats
	if err := r.db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT NULLIF(user_id, '')) FROM track_event WHERE event_type = 'PAGEVIEW'`).
		Scan(&s.TotalPV, &s.TotalUV); err != nil {
		return s, err
	}
	if err := r.db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT NULLIF(user_id, '')) FROM track_event WHERE created_at >= CURRENT_DATE AND event_type = 'PAGEVIEW'`).
		Scan(&s.TodayPV, &s.TodayUV); err != nil {
		return s, err
	}
	if err := r.db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT NULLIF(user_id, '')) FROM track_event WHERE created_at >= CURRENT_DATE - INTERVAL '1 day' AND created_at < CURRENT_DATE AND event_type = 'PAGEVIEW'`).
		Scan(&s.YesterdayPV, &s.YesterdayUV); err != nil {
		return s, err
	}
	if err := r.db.QueryRow(`SELECT COUNT(DISTINCT NULLIF(user_id, '')) FROM track_event WHERE created_at >= NOW() - INTERVAL '5 minutes' AND event_type = 'PAGEVIEW'`).
		Scan(&s.OnlineUsers); err != nil {
		return s, err
	}
	if err := r.db.QueryRow(`SELECT ROUND(COALESCE(AVG(event_duration), 0))::bigint FROM track_event WHERE event_type = 'PAGE_LEAVE' AND event_duration > 0 AND event_duration <= 3600`).
		Scan(&s.AverageReadSeconds); err != nil {
		return s, err
	}
	return s, nil
}

// === 读取操作：趋势 ===

// DailyStat 单日统计
type DailyStat struct {
	Date string
	PV   int64
	UV   int64
}

func (r *TrackingRepo) DailyStats(days int) ([]DailyStat, error) {
	if days < 1 {
		days = 7
	}
	if days > 365 {
		days = 365
	}
	rows, err := r.db.Query(`
		WITH dates AS (
			SELECT generate_series(
				CURRENT_DATE - (($1 - 1) * INTERVAL '1 day'),
				CURRENT_DATE,
				INTERVAL '1 day'
			)::date AS day
		)
		SELECT TO_CHAR(d.day, 'YYYY-MM-DD') AS date,
			COUNT(e.id) AS pv,
			COUNT(DISTINCT NULLIF(e.user_id, '')) AS uv
		FROM dates d
		LEFT JOIN track_event e
			ON e.created_at >= d.day
			AND e.created_at < d.day + INTERVAL '1 day'
			AND e.event_type = 'PAGEVIEW'
		GROUP BY d.day
		ORDER BY d.day ASC
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSlice(rows, func(d *DailyStat) error {
		return rows.Scan(&d.Date, &d.PV, &d.UV)
	})
}

// === 读取操作：热门页面 ===

// PageStat 页面统计
type PageStat struct {
	Path  string
	Title string
	PV    int64
	UV    int64
}

func (r *TrackingRepo) TopPages(limit int) ([]PageStat, error) {
	rows, err := r.db.Query(`
		SELECT page_path,
			COALESCE(MAX(NULLIF(metadata->>'title', '')), page_path) AS title,
			COUNT(*) as pv,
			COUNT(DISTINCT NULLIF(user_id, '')) as uv
		FROM track_event
		WHERE (page_path LIKE '/tech/%' OR page_path LIKE '/life/%')
		  AND page_path NOT LIKE '%?%' AND event_type = 'PAGEVIEW'
		GROUP BY page_path ORDER BY pv DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PageStat
	for rows.Next() {
		var p PageStat
		if err := rows.Scan(&p.Path, &p.Title, &p.PV, &p.UV); err != nil {
			return nil, err
		}
		if decoded, err := url.PathUnescape(p.Path); err == nil {
			p.Path = decoded
		}
		p.Path = strings.TrimSuffix(p.Path, ".html")
		if p.Title == "" {
			p.Title = p.Path
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

// === 读取操作：分类统计 ===

// CategoryStat 分类统计
type CategoryStat struct {
	Name  string
	Value int64
}

// AllowedCategoryColumns 允许的分类统计列名白名单
var AllowedCategoryColumns = map[string]bool{
	"device_type": true,
	"platform":    true,
}

func (r *TrackingRepo) CategoryStats(column string) ([]CategoryStat, error) {
	if !AllowedCategoryColumns[column] {
		return nil, nil
	}
	rows, err := r.db.Query(
		"SELECT " + column + ", COUNT(*) as count FROM track_event WHERE event_type = 'PAGEVIEW' AND " + column + " IS NOT NULL AND " + column + " != '' GROUP BY " + column + " ORDER BY count DESC LIMIT 10",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSlice(rows, func(c *CategoryStat) error {
		return rows.Scan(&c.Name, &c.Value)
	})
}

func (r *TrackingRepo) MetadataStats(key string) ([]CategoryStat, error) {
	rows, err := r.db.Query(`
		SELECT metadata->>$1 as name, COUNT(*) as count
		FROM track_event
		WHERE event_type = 'PAGEVIEW' AND metadata->>$1 IS NOT NULL AND metadata->>$1 != ''
		GROUP BY name ORDER BY count DESC LIMIT 10
	`, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSlice(rows, func(c *CategoryStat) error {
		return rows.Scan(&c.Name, &c.Value)
	})
}

// === 工具 ===

// CountAll 返回总事件数
func (r *TrackingRepo) CountAll() (int64, error) {
	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM track_event").Scan(&count)
	return count, err
}

// TrackEventRow 埋点事件行（用于批量插入）
type TrackEventRow struct {
	SessionID, UserID, EventType, ElementPath, PagePath, Referrer string
	Metadata, UserAgent, IPAddress                                string
	CreatedAt                                                     time.Time
	CustomProperties, Platform, DeviceInfo                        string
	EventDuration                                                 int
	DeviceID, Version, DeviceType                                 string
}

func (e *TrackEventRow) defaults() {
	if e.Metadata == "" {
		e.Metadata = "{}"
	}
	if e.CustomProperties == "" {
		e.CustomProperties = "{}"
	}
	if e.DeviceInfo == "" {
		e.DeviceInfo = "{}"
	}
}

// scanSlice 通用 rows → slice 转换
func scanSlice[T any](rows *sql.Rows, scanFn func(*T) error) ([]T, error) {
	var results []T
	for rows.Next() {
		var item T
		if err := scanFn(&item); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}
